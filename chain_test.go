// Package alice implements a middleware chaining solution.
package alice

import (
	"reflect"
	"testing"

	"bufio"

	"github.com/valyala/fasthttp"

	"bytes"
)

// A constructor for middleware
// that writes its own "tag" into the RW and does nothing else.
// Useful in checking if a chain is behaving in the right order.
func tagMiddleware(tag string) Constructor {
	return func(h fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			ctx.Write([]byte(tag))
			h(ctx)
		}
	}
}

// Not recommended (https://golang.org/pkg/reflect/#Value.Pointer),
// but the best we can do.
func funcsEqual(f1, f2 interface{}) bool {
	val1 := reflect.ValueOf(f1)
	val2 := reflect.ValueOf(f2)
	return val1.Pointer() == val2.Pointer()
}

var testApp = func(ctx *fasthttp.RequestCtx) {
	ctx.Write([]byte("app\n"))
}

func TestNew(t *testing.T) {
	c1 := func(h fasthttp.RequestHandler) fasthttp.RequestHandler {
		return nil
	}

	c2 := func(h fasthttp.RequestHandler) fasthttp.RequestHandler {
		return nil
	}

	slice := []Constructor{c1, c2}

	chain := New(slice...)
	for k := range slice {
		if !funcsEqual(chain.constructors[k], slice[k]) {
			t.Error("New does not add constructors correctly")
		}
	}
}

func TestThenWorksWithNoMiddleware(t *testing.T) {
	if !funcsEqual(New().Then(testApp), testApp) {
		t.Error("Then does not work with no middleware")
	}
}

func TestThenFuncConstructsHandlerFunc(t *testing.T) {
	fn := func(ctx *fasthttp.RequestCtx) {
		ctx.Response.SetStatusCode(200)
	}
	chained := New().Then(fn)

	var ctx fasthttp.RequestCtx
	chained(&ctx)

	if reflect.TypeOf(chained) != reflect.TypeOf((fasthttp.RequestHandler)(nil)) {
		t.Error("ThenFunc does not construct HandlerFunc")
	}
}
func newGetCtx(t *testing.T) *fasthttp.RequestCtx {
	var ctx fasthttp.RequestCtx

	s := "GET / HTTP/1.1\nHost: aaa.com\n\n"
	br := bufio.NewReader(bytes.NewBufferString(s))
	if err := ctx.Request.Read(br); err != nil {
		t.Fatalf("cannot read request: %s", err)
	}
	return &ctx
}
func TestThenOrdersHandlersCorrectly(t *testing.T) {
	t1 := tagMiddleware("t1\n")
	t2 := tagMiddleware("t2\n")
	t3 := tagMiddleware("t3\n")

	chained := New(t1, t2, t3).Then(testApp)
	ctx := newGetCtx(t)

	chained(ctx)
	if string(ctx.Response.Body()) != "t1\nt2\nt3\napp\n" {
		t.Error("Then does not order handlers correctly")
	}
}

func TestAppendAddsHandlersCorrectly(t *testing.T) {
	chain := New(tagMiddleware("t1\n"), tagMiddleware("t2\n"))
	newChain := chain.Append(tagMiddleware("t3\n"), tagMiddleware("t4\n"))

	if len(chain.constructors) != 2 {
		t.Error("chain should have 2 constructors")
	}
	if len(newChain.constructors) != 4 {
		t.Error("newChain should have 4 constructors")
	}

	chained := newChain.Then(testApp)

	ctx := newGetCtx(t)

	chained(ctx)

	if string(ctx.Response.Body()) != "t1\nt2\nt3\nt4\napp\n" {
		t.Error("Append does not add handlers correctly")
	}
}

func TestAppendRespectsImmutability(t *testing.T) {
	chain := New(tagMiddleware(""))
	newChain := chain.Append(tagMiddleware(""))

	if &chain.constructors[0] == &newChain.constructors[0] {
		t.Error("Apppend does not respect immutability")
	}
}

func TestExtendAddsHandlersCorrectly(t *testing.T) {
	chain1 := New(tagMiddleware("t1\n"), tagMiddleware("t2\n"))
	chain2 := New(tagMiddleware("t3\n"), tagMiddleware("t4\n"))
	newChain := chain1.Extend(chain2)

	if len(chain1.constructors) != 2 {
		t.Error("chain1 should contain 2 constructors")
	}
	if len(chain2.constructors) != 2 {
		t.Error("chain2 should contain 2 constructors")
	}
	if len(newChain.constructors) != 4 {
		t.Error("newChain should contain 4 constructors")
	}

	chained := newChain.Then(testApp)

	ctx := newGetCtx(t)

	chained(ctx)

	if string(ctx.Response.Body()) != "t1\nt2\nt3\nt4\napp\n" {
		t.Error("Extend does not add handlers in correctly")
	}
}

func TestExtendRespectsImmutability(t *testing.T) {
	chain := New(tagMiddleware(""))
	newChain := chain.Extend(New(tagMiddleware("")))

	if &chain.constructors[0] == &newChain.constructors[0] {
		t.Error("Extend does not respect immutability")
	}
}
