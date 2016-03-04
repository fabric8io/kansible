package env

import (
	"fmt"
	"os"

	"github.com/Masterminds/cookoo"
	"github.com/Masterminds/cookoo/log"
)

// Get gets one or more environment variables and puts them into the context.
//
// Parameters passed in are of the form varname => defaultValue.
//
// 	r.Route("foo", "example").Does(envvar.Get).Using("HOME").WithDefault(".")
//
// As with all environment variables, the default value must be a string.
//
// WARNING: Since parameters are a map, order of processing is not
// guaranteed. If order is important, you'll need to call this command
// multiple times.
//
// For each parameter (`Using` clause), this command will look into the
// environment for a matching variable. If it finds one, it will add that
// variable to the context. If it does not find one, it will expand the
// default value (so you can set a default to something like "$HOST:$PORT")
// and also put the (unexpanded) default value back into the context in case
// any subsequent call to `os.Getenv` occurs.
func Get(c cookoo.Context, params *cookoo.Params) (interface{}, cookoo.Interrupt) {
	for name, def := range params.AsMap() {
		var val string
		if val = os.Getenv(name); len(val) == 0 {
			if def == nil {
				def = ""
			}
			def, ok := def.(string)
			if !ok {
				log.Warnf(c, "Could not convert %s. Type is %T", name, def)
			}
			val = os.ExpandEnv(def)
			// We want to make sure that any subsequent calls to Getenv
			// return the same default.
			os.Setenv(name, val)

		}
		c.Put(name, val)
		log.Debugf(c, "Name: %s, Val: %s", name, val)
	}
	return true, nil
}

// Expand expands the environment variables in the given string and returns the result.
//
// Params:
// 	- content (string): The given string to expand.
//
// Returns:
//  - The expanded string. This expands against the os environemnt (os.ExpandEnv).
func Expand(c cookoo.Context, p *cookoo.Params) (interface{}, cookoo.Interrupt) {
	s := p.Get("content", "").(string)

	// TODO: We could easily add support here for Expand().

	return os.ExpandEnv(s), nil
}

// Set takes the given names and values and puts them into both the context
// and the environment.
//
// Unlike Get, it does not try to retrieve the values from the environment
// first.
//
// Values are passed through os.ExpandEnv()
//
// There is no guarantee of insertion order. If multiple name/value pairs
// are given, they will be put into the context in whatever order they
// are retrieved from the underlying map.
//
// Params:
//   accessed as map[string]string
// Returns:
//   nothing, but inserts all name/value pairs into the context and the
//   environment.
func Set(c cookoo.Context, p *cookoo.Params) (interface{}, cookoo.Interrupt) {
	for name, def := range p.AsMap() {
		// Assume Nil means unset the value.
		if def == nil {
			def = ""
		}

		val := fmt.Sprintf("%v", def)
		val = os.ExpandEnv(val)
		log.Debugf(c, "Name: %s, Val: %s", name, val)

		os.Setenv(name, val)
		c.Put(name, val)
	}
	return true, nil

}
