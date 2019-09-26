package cfg

import (
	"flag"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"reflect"
	"strings"
)

type argInfo struct {
	argPtr interface{}
	set    bool
}

func registerTypes(moduleOptions interface{}, argPointers map[string]*argInfo) error {
	moduleOptionsType := reflect.TypeOf(moduleOptions).Elem()

	numFields := moduleOptionsType.NumField()

	moduleOptionsValue := reflect.ValueOf(moduleOptions).Elem()

	// register flags for the module specific options
	for i := 0; i < numFields; i++ {
		field := moduleOptionsType.Field(i)

		if field.PkgPath != "" {
			continue
		}

		fieldValue := moduleOptionsValue.Field(i)

		cliName, found := field.Tag.Lookup("cli")
		if !found {
			continue
		}

		cliDescription, found := field.Tag.Lookup("desc")
		if !found {
			cliDescription = ""
		}

		switch fieldValue.Interface().(type) {
		case string:
			strRef := flag.String(cliName, "", cliDescription)
			argPointers[cliName] = &argInfo{argPtr: strRef}
		case []string:
			strRef := flag.String(cliName, "", cliDescription)
			argPointers[cliName] = &argInfo{argPtr: strRef}
		case bool:
			boolRef := flag.Bool(cliName, false, cliDescription)
			argPointers[cliName] = &argInfo{argPtr: boolRef}
		default:
			return fmt.Errorf("type %s not handled", field.Type)
		}
	}

	return nil
}

func checkForSet(argPointers map[string]*argInfo) {
	flag.Visit(func(f *flag.Flag) {
		if _, found := argPointers[f.Name]; found {
			argPointers[f.Name].set = true
		}
	})
}

func fillOptionsWithMap(options interface{}, argPointers map[string]*argInfo) {
	t := reflect.TypeOf(options).Elem()
	v := reflect.ValueOf(options).Elem()
	for i := 0; i < t.NumField(); i++ {
		ft := t.Field(i)
		cliName, found := ft.Tag.Lookup("cli")
		if !found {
			continue
		}

		fv := v.Field(i)

		if ai, found := argPointers[cliName]; found && ai.set {
			typeInterface := fv.Interface()

			switch typeInterface.(type) {
			case string:
				fv.SetString(*ai.argPtr.(*string))
			case []string:
				argsStr := *ai.argPtr.(*string)
				args := strings.Split(argsStr, ",")

				fv.Set(reflect.ValueOf(args))
			case bool:
				fv.SetBool(*ai.argPtr.(*bool))
			}
		}
	}
}

// LoadFlags loads 2 sets of options: global options defined by the GlobalOptions struct
// and local options provided by the passed moduleOptions parameter.
func LoadFlags(moduleOptions interface{}, globalOptions *GlobalOptions) error {
	argPointers := make(map[string]*argInfo)

	if err := registerTypes(moduleOptions, argPointers); err != nil {
		return err
	}
	if err := registerTypes(globalOptions, argPointers); err != nil {
		return err
	}
	flag.Parse()

	checkForSet(argPointers)

	// special config key needed for loading config file
	// this loads everything from the config file
	if ap, found := argPointers["config"]; found && ap.set {
		configFile := ap.argPtr.(*string)

		configBytes, err := ioutil.ReadFile(*configFile)
		if err != nil {
			return err
		}

		err = yaml.Unmarshal(configBytes, globalOptions)
		if err != nil {
			return err
		}

		err = yaml.Unmarshal(configBytes, moduleOptions)
		if err != nil {
			return err
		}
	}

	fillOptionsWithMap(moduleOptions, argPointers)
	fillOptionsWithMap(globalOptions, argPointers)

	return nil
}
