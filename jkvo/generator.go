package jkvo;

import(
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"
)


type KvoTemplateParams struct {
	Package string
	OuterClass string
	Types []KvoType
}

var kvoTemplate = template.Must(template.New("kvo").Parse(`
package {{.Package}};

import ch.andrewreus.kvo.KvoObject;
import java.util.LinkedList;

public class {{.OuterClass}} {

{{range $kvoType := .Types}}
public static class {{$kvoType.Name}} extends KvoObject {
{{range $prop := $kvoType.Properties}}
    // Key "{{$prop.Key}}"
    public interface {{$prop.CamelCaseKey}}Subscriber {
        public void didUpdate{{$prop.CamelCaseKey}}({{$prop.Type}} old);
    }

    private KvoObject.KvoProperty<{{$prop.Type}}> prop{{$prop.CamelCaseKey}}_ = new KvoObject.KvoProperty<{{$prop.Type}}>({{$prop.InitialValue}}) {

        protected void notifySubscribers({{$prop.Type}} old) {
            notify{{$prop.CamelCaseKey}}Subscribers(old);
        }
    };

    public LinkedList<{{$prop.CamelCaseKey}}Subscriber> subscribersTo{{$prop.CamelCaseKey}} = new LinkedList<{{$prop.CamelCaseKey}}Subscriber>();

    private void notify{{$prop.CamelCaseKey}}Subscribers({{$prop.Type}} old) {
        for ({{$prop.CamelCaseKey}}Subscriber s : subscribersTo{{$prop.CamelCaseKey}}) {
            s.didUpdate{{$prop.CamelCaseKey}}(old);
        }
    }

    public {{$prop.Type}} get{{$prop.CamelCaseKey}}() {
        return prop{{$prop.CamelCaseKey}}_.get();
    }

    public void subscribeTo{{$prop.CamelCaseKey}}({{$prop.CamelCaseKey}}Subscriber subscriber) {
        subscribersTo{{$prop.CamelCaseKey}}.add(subscriber);
    }

    public void set{{$prop.CamelCaseKey}}({{$prop.Type}} value) {
        prop{{$prop.CamelCaseKey}}_.update(value);
    }
{{end}}
}
{{end}}
}
`))

type KvoObject map[string]interface{}

type ValidationError struct {
	Problem string
	Field string
}

func (v ValidationError) Error() string {
	if v.Field == "" {
		return v.Problem
	} else {
		return "In field " + v.Field + ": " + v.Problem
	}
}

type KvoProperty struct {
	Key string
	Type string
	InitialValue string
	IsExternal bool
}

func (p KvoProperty) IsInitialized() bool {
	return p.Key != "" && p.Type != "" && (p.IsComplex() || p.IsExternal || p.InitialValue != "")
}

var primitiveTypes []string = []string{"Boolean", "Integer", "Double", "String"}

func (p KvoProperty) IsComplex() bool {
	for _, pt := range primitiveTypes {
		if p.Type == pt {
			return false
		}
	}
	return true
}

type KvoType struct {
	Name string
	Properties []KvoProperty
}

func (k KvoProperty) CamelCaseKey() string {
	return strings.ToUpper(k.Key[0:1]) + k.Key[1:]
}

func ParseAndValidate(r io.Reader) (obj KvoObject, err error) {
	d := json.NewDecoder(r)
	d.UseNumber()  // Do this so we can detect "0.0" floats.
	if err = d.Decode(&obj); err != nil {
		return
	}
	return
}

func SpecEntryToProperty(key string, value interface{}, enclosingType string) (prop KvoProperty) {
	prop.Key = key
	switch value.(type) {
	case map[string]interface{}:
		// The value is either a sub-field or an external type.
		mapValue := value.(map[string]interface{})
		if mapTypeName, ok := mapValue["__name__"]; ok {
			prop.Type = mapTypeName.(string)
			if javaPkg, ok := mapValue["__package__"]; ok {
				prop.Type = javaPkg.(string) + "." + prop.Type
  			prop.IsExternal = true
			}
		} else {
			prop.Type = enclosingType + "_" + key
		}
		if !prop.IsExternal {
			prop.InitialValue = "new " + prop.Type + "()";
		} else {
			prop.InitialValue = "null";
		}
	case bool:
		prop.Type = "Boolean"
		if value.(bool) {
			prop.InitialValue = "true"
		} else {
			prop.InitialValue = "false"
		}
	case json.Number:
		numValue := value.(json.Number)
		if strings.Index(numValue.String(), ".") != -1 {
			prop.Type = "Double"
			prop.InitialValue = numValue.String()
		} else {
			prop.Type = "Integer"
			prop.InitialValue = numValue.String()
		}
	case string:
		prop.Type = "String"
		prop.InitialValue = value.(string)
	}
	return
}

func TypeToVarList(typeName string, obj KvoObject, types *[]KvoType) error {
	kt := KvoType{Name: typeName, Properties: make([]KvoProperty, 0, len(obj))}
	for k, t := range obj {
		if k == "__name__" {
			continue
		}
		prop := SpecEntryToProperty(k, t, typeName)
		if !prop.IsInitialized() {
			return ValidationError{
			Problem: fmt.Sprintf("Don't know how to generate for value %#v", t),
			Field: typeName + "_" + k}
		}
		if !prop.IsExternal && prop.IsComplex() {
			if err := TypeToVarList(prop.Type, t.(map[string]interface{}), types); err != nil {
				return err
			}
		}
		kt.Properties = append(kt.Properties, prop)
	}
	*types = append(*types, kt)
	return nil
}

func Generate(javaPkg string, outerClass string, o map[string]interface{}, w io.Writer) (error) {
	if javaPkg == "" {
		return ValidationError{Problem: "Must specify a Java package.", Field: ""}
	}
	params := KvoTemplateParams{Package: javaPkg, OuterClass: outerClass, Types: make([]KvoType, 0)}
	prop := SpecEntryToProperty("", o, "")
	if prop.Type == "_" {
		return ValidationError{Problem: "Outermost object must specify a type name with key __name__", Field: ""}
	}
	if err := TypeToVarList(prop.Type, o, &params.Types); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Params: %v\n", params)

	if err := kvoTemplate.Execute(w, params); err != nil {
 		return err
	}
	return nil
}
