package route

import (
	"github.com/revel/revel"
//	"github.com/leanote/leanote/app/service"
//	. "github.com/leanote/leanote/app/lea"
	"net/url"
	"strings"
)

// overwite revel RouterFilter
// /api/user/Info => ApiUser.Info()
var staticPrefix = []string{"/public", "/favicon.ico", "/css", "/js", "/images", "/tinymce", "/upload", "/fonts"}
func RouterFilter(c *revel.Controller, fc []revel.Filter) {
	// 补全controller部分
	path := c.Request.Request.URL.Path
	
	// Figure out the Controller/Action
	var route *revel.RouteMatch = revel.MainRouter.Route(c.Request.Request)
	if route == nil {
		c.Result = c.NotFound("No matching route found: " + c.Request.RequestURI)
		return
	}

	// The route may want to explicitly return a 404.
	if route.Action == "404" {
		c.Result = c.NotFound("(intentionally)")
		return
	}
	
	//----------
	// life start
	/*
	type URL struct {
	    Scheme   string
	    Opaque   string    // encoded opaque data
	    User     *Userinfo // username and password information
	    Host     string    // host or host:port
	    Path     string
	    RawQuery string // encoded query values, without '?'
	    Fragment string // fragment for references, without '#'
	}
	*/
	if route.ControllerName != "Static" {
		// api设置
		// leanote.com/api/user/get => ApiUser::Get
		if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "api/") {
			route.ControllerName = "Api" + route.ControllerName
		}
		// end
	}
	
	// Set the action.
	if err := c.SetAction(route.ControllerName, route.MethodName); err != nil {
		c.Result = c.NotFound(err.Error())
		return
	}

	// Add the route and fixed params to the Request Params.
	c.Params.Route = route.Params

	// Add the fixed parameters mapped by name.
	// TODO: Pre-calculate this mapping.
	for i, value := range route.FixedParams {
		if c.Params.Fixed == nil {
			c.Params.Fixed = make(url.Values)
		}
		if i < len(c.MethodType.Args) {
			arg := c.MethodType.Args[i]
			c.Params.Fixed.Set(arg.Name, value)
		} else {
			revel.WARN.Println("Too many parameters to", route.Action, "trying to add", value)
			break
		}
	}

	fc[0](c, fc[1:])
}

