package envflag

import (
	"flag"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"
)

type errors struct {
	errs []error
}

func (e *errors) add(err error) {
	if err == nil {
		return
	}
	e.errs = append(e.errs, err)
}

func (e *errors) has() bool {
	return len(e.errs) > 0
}

func (e *errors) get() error {
	msgs := make([]string, len(e.errs))
	for i, err := range e.errs {
		msgs[i] = err.Error()
	}
	return fmt.Errorf(strings.Join(msgs, "\n"))
}

// Parameter describes a configurable part of the application.
type Parameter struct {

	// Key identifies the parameter.
	// It is unique per Parameter managed by a FlagManager.
	Key string `json:"key"`

	// Type is the type of the parameter.
	Type reflect.Type `json:"type"`

	// EnvKey is the name of the environment variable configuring this parameter.
	EnvKey string `json:"env"`

	// The ArgKey is the name of the command line argument configuring this parameter.
	ArgKey string `json:"arg"`

	// ArgAliases are alternatives for ArgKey.
	ArgAliases []string `json:"argalt"`

	// Value is the current value in string form.
	Value string `json:"value"`

	// DefaultValue is the default value in string form.
	DefaultValue string `json:"default"`

	// Options contains all values the parameter can take.
	// If the value is not an Enumerator, it is empty.
	Options []ParameterValue `json:"options"`

	// Tag is an optional tag for this parameter.
	// It can be used to only show important parameters in short help texts.
	Tag string `json:"tag"`

	Description string `json:"desc"`
}

// ParameterValue describes one possible value a Parameter can take.
type ParameterValue struct {
	Value       string `json:"value"`
	Description string `json:"desc"`
}

// Vars is a pointer to a struct containing configuration variables.
// Struct tags can be used to configure the behavior of parameters,
// all tags are optional.
//
//     type Config struct {
//	       a string `key:"override the key, otherwise it is the field name"`
//	       b string `args:"comma separated alternative command line arg representations"`
//	       c string `desc:"a description of what the parameter does"`
//	       d string `tag:"a tag useable for filtering, e.g. when generating documentation"`
//     }
//
// In addition to the tag based configuration, the field name and type are used and
// the current value on registration is used as the default value.
type Vars interface{}

// Value is the interface to the dynamic value stored in a flag. (The default value is represented as a string.)
type Value interface {
	String() string
	Set(string) error
}

// Value is a flag.Value copied here to avoid an import dependency for users.
// Assert type compatibility:
var v Value
var _ flag.Value = v

// Enumerator is a Value from an enumerable short set of distinct values.
// Its main use is for the equivalent of an enum of strings.
type Enumerator interface {
	Value
	Values() []string
	Describe(value string) string
}

// Env is a configuration environment grouped by a common variable prefix.
type Env struct {
	prefix string
}

func Environment(prefix string) Env {
	return Env{prefix: prefix}
}

var (
	invalidchars = regexp.MustCompile("[^A-Za-z0-9_]+")
	uncamel      = regexp.MustCompile("([A-Z])")
	leadingdash  = regexp.MustCompile("^-+")
)

func (e Env) keyToAny(key string) string {
	return uncamel.ReplaceAllString(key, "-$1")
}

func (e Env) keyToArg(key string) string {
	key = e.keyToAny(key)
	key = invalidchars.ReplaceAllLiteralString(key, "-")
	key = leadingdash.ReplaceAllLiteralString(key, "")
	return strings.ToLower(key)
}

func (e Env) keyToEnv(key string) string {
	key = e.keyToAny(e.prefix + key)
	key = invalidchars.ReplaceAllLiteralString(key, "_")
	return strings.ToUpper(key)
}

// WithParameters creates a group of managed parameters.
func (e Env) WithParameters(name string) Parameters {
	mgr := &parameters{
		Env:    e,
		name:   name,
		values: make(map[string]*reference),
	}
	mgr.Init(name, flag.ContinueOnError)
	mgr.Usage = func() {} // disable native FlagSet output
	return mgr
}

// Parameters manages struct fields as configuration parameters and enables their configuration
// from different sources, e.g. command line arguments and environment variables.
//
// Each parameter is represented by a key. The key name is provided with a struct tag or
// matches the field name.
//
// A parameter may be modified by command line arguments (ARG) or an environment variable (ENV).
// The key is used to derive the primary ARG and the ENV:
//
// ARG and ENV can only contain English letters, digits and a separator (ENV: '_', ARG: '-').
// All other characters in the key are replaced with separators.
// Upper case letters are lower cased and prefixed with a separator.
// ENV is prefixed with the Environment prefix.
// Multiple separators are reduced to one.
// Leading separators are removed, ARG may be used prefixed with one or two separators.
// Then, ENV is upper cased, ARG is lower cased.
//
// Examples for Environment prefix "myapp-":
//     Key      ARG        ENV
//     -------|----------|--------------
//     MyKey	my-key     MYAPP_MY_KEY
//     Val      val        MYAPP_VAL
//     Über     ber        MYAPP_BER
//
// Usage:
//     # ARG with single leading dash
//     myapp -my-key=Value
//
//     # ARG with double leading dash
//     myapp --my-key=Value2
//
//     # ENV
//     export MYAPP_VAL=value
//     myapp
type Parameters interface {

	// Register registers struct fields as configuration parameters.
	//
	// It must be called with a non-nil struct pointer and panics otherwise.
	// The current values of each field are used as default values.
	Register(vars Vars)

	// Keys retrieves a slice of parameter keys for all managed parameters.
	Keys() []string

	// ArgKey retrieves the command line argument used to configure the parameter
	// identified by the given key.
	ArgKey(key string) string

	// ArgAliases retrives a slice of alternative command line arguments also useable
	// to configure the parameter identified by the given key.
	ArgAliases(key string) []string

	// EnvKey retrieves the name of the environment variable used to configure the
	// parameter identified by the given key.
	EnvKey(key string) string

	// SetValues calls a function for every managed parameter with its EnvKey.
	// It sets the parameter to the value returned by the function if the call to
	// Set on the Value does not return an error.
	//
	// To set the default values from environment variables, the argument should be
	//     os.Getenv
	SetValues(func(string) string) error

	// Parse parses parameter definitions from the argument list, which should not
	// include the command name.
	//
	// Must be called after all parameters are registered and before they are accessed
	// by the program.
	Parse(args []string) error

	// ArgRest retrieves all unparsed parameters.
	ArgRest() []string

	// Explore retrieves a slice of all managed parameters with additional information.
	// Use Explore as the central source to generate documentation.
	Explore() []Parameter
}

type parameters struct {
	Env
	flag.FlagSet
	name   string
	values map[string]*reference
}

type reference struct {
	base    interface{}
	ptr     interface{}
	name    string
	arg     string
	tag     string
	aliases []string
}

func (ps *parameters) Register(vars Vars) {
	if vars == nil {
		return
	}
	pv := reflect.ValueOf(vars)
	for pv.Kind() == reflect.Ptr {
		pv = pv.Elem()
	}
	pt := pv.Type()
	if pt.Kind() != reflect.Struct {
		panic(fmt.Errorf("%T must be a *struct", vars))
	}
	errs := &errors{}
	if pt.Kind() != reflect.Struct {
		panic(fmt.Errorf("%T must be a *struct", vars))
	}
	for i, numFields := 0, pt.NumField(); i < numFields; i++ {
		field := pt.Field(i)
		value := pv.Field(i)
		valueptr := value.Addr().Interface()
		name, key, desc, tag, rawargs := parseField(&field)
		var refarg string
		var aliases []string
		for j, raw := range rawargs {
			arg := ps.keyToArg(raw)
			switch val := valueptr.(type) {
			case *bool:
				ps.BoolVar(val, arg, *val, desc)
			case *int:
				ps.IntVar(val, arg, *val, desc)
			case *int64:
				ps.Int64Var(val, arg, *val, desc)
			case *uint:
				ps.UintVar(val, arg, *val, desc)
			case *uint64:
				ps.Uint64Var(val, arg, *val, desc)
			case *string:
				ps.StringVar(val, arg, *val, desc)
			case *time.Duration:
				ps.DurationVar(val, arg, *val, desc)
			default:
				paramVal, ok := value.Interface().(flag.Value)
				if !ok {
					err := fmt.Errorf(
						"type error in %T: %q must implement Value",
						vars, name,
					)
					errs.add(err)
					continue
				}
				ps.Var(paramVal, arg, desc)
			}
			if j == 0 {
				refarg = arg
				desc = "-> alias for -" + arg
			} else {
				aliases = append(aliases, arg)
			}
		}
		ps.values[key] = &reference{
			base:    vars,
			ptr:     valueptr,
			name:    name,
			arg:     refarg,
			tag:     tag,
			aliases: aliases,
		}
	}
	if !errs.has() {
		return
	}
	// Errors landing here can only be caused by a type error.
	// They are development specific and fixable - make them visible!
	panic(errs.get())
}

func parseField(field *reflect.StructField) (name, key, desc, tag string, args []string) {
	name = field.Name
	paramTag := field.Tag
	key = paramTag.Get("key")
	if key == "" {
		key = name
	}
	args = []string{key}
	if rawargs := paramTag.Get("args"); rawargs != "" {
		args = append(args, strings.Split(rawargs, ",")...)
	}
	desc = paramTag.Get("desc")
	tag = paramTag.Get("tag")
	return
}

func (ps *parameters) Keys() []string {
	keys := make([]string, 0, len(ps.values))
	for k, _ := range ps.values {
		keys = append(keys, k)
	}
	return keys
}

func (ps *parameters) ArgKey(key string) string {
	val, ok := ps.values[key]
	if !ok {
		return ""
	}
	return val.arg
}

func (ps *parameters) ArgAliases(key string) []string {
	return append([]string{}, ps.values[key].aliases...)
}

func (ps *parameters) EnvKey(key string) string {
	_, ok := ps.values[key]
	if !ok {
		return ""
	}
	return ps.keyToEnv(key)
}

func (ps *parameters) SetDefaults(env func(string) string) error {
	errs := &errors{}
	for k, v := range ps.values {
		val := env(ps.keyToEnv(k))
		if val != "" {
			errs.add(ps.Set(v.arg, val))
		}
	}
	if errs.has() {
		return errs.get()
	}
	return nil
}

func (ps *parameters) Parse(args []string) error {
	err := ps.FlagSet.Parse(args)
	if err == flag.ErrHelp {
		return nil
	}
	return err
}

func (ps *parameters) ArgRest() []string {
	return ps.FlagSet.Args()
}

func (ps *parameters) Explore() []Parameter {
	params := make([]Parameter, len(ps.values))
	i := 0
	for key, v := range ps.values {
		p := &params[i]
		i++
		pflag := ps.Lookup(v.arg)
		p.Key = key
		p.Type = reflect.TypeOf(v.ptr).Elem()
		p.EnvKey = ps.keyToEnv(key)
		p.ArgKey = v.arg
		p.ArgAliases = append([]string{}, v.aliases...)
		p.Value = pflag.Value.String()
		p.DefaultValue = pflag.DefValue
		p.Description = pflag.Usage
		p.Tag = v.tag
		if enum, ok := pflag.Value.(Enumerator); ok {
			values := enum.Values()
			p.Options = make([]ParameterValue, len(values))
			for i, value := range values {
				p.Options[i] = ParameterValue{
					Value:       value,
					Description: enum.Describe(value),
				}
			}
		}
	}
	return params
}
