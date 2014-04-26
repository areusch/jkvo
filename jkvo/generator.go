package jkvo;

import(
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"text/template"
)


type KvoTemplateParams struct {
	Package string
	Types []KvoType
}

var kvoTemplate = template.Must(template.New("kvo").Parse(`
package {{.Package}};

import ch.andrewreus.kvo.KvoObject;

{{range $kvoType := .Types}}
public class {{$kvoType.Name}} extends KvoObject {
{{range $prop := $kvoType.Properties}}
    // Key "{{$prop.Key}}"
    public interface {{$prop.CamelCaseKey}}Subscriber {
        public void didUpdate{{$prop.CamelCaseKey}}({{$prop.Type}} old);
    }

    public KvoProperty<{{$prop.Type}}> prop{{$prop.CamelCaseKey}}_ = KvoProperty<{{$prop.Type}}>({{$prop.InitialValue}}) {
        LinkedList<{{$prop.CamelCaseKey}}Subscriber> subscribers_ = new LinkedList<{{$prop.CamelCaseKey}}>();

        protected void notifySubscribers({{$prop.Type}} old) {
            for (Prop{{$prop.CamelCaseKey}}Subscriber s : subscribers_) {
                s.didUpdate{{$prop.CamelCaseKey}}(old);
            }
        }
    };

    public {{$prop.Type}} get{{$prop.CamelCaseKey}}() {
        return prop{{$prop.CamelCaseKey}}_.get();
    }

    public void subscribeTo{{$prop.CamelCaseKey}}(Prop{{$prop.CamelCaseKey}}Listener subscriber) {
        prop{{$prop.CamelCaseKey}}_.addSubscriber(subscriber);
    }

    public set{{$prop.CamelCaseKey}}({{$prop.Type}} value) {
        prop{{$prop.CamelCaseKey}}_.set(value);
    }
{{end}}
}
{{end}}
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
}

type KvoType struct {
	Name string
	Properties []KvoProperty
}

func (k KvoProperty) CamelCaseKey() string {
	return strings.ToUpper(k.Key[0:1]) + k.Key[1:]
}

func ParseAndValidate(r io.Reader) (obj KvoObject, err error) {
	if err = json.NewDecoder(r).Decode(&obj); err != nil {
		return
	}
	return
}

func TypeNameFromKind(k reflect.Kind, enclosingType string, propName string, val interface{}) (typeName string, initialValue string, isComplex bool) {
	switch k {
	case reflect.Map:
		mapVal := val.(map[string]interface{})
		if mapTypeName, ok := mapVal["__name__"]; ok {
			typeName = mapTypeName.(string)
		} else {
			typeName = enclosingType + "_" + propName
		}
		initialValue = "new " + typeName + "()";
		isComplex = true
	case reflect.Bool:
		typeName = "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		typeName = "int"
		initialValue = "0";
	case reflect.Float32, reflect.Float64:
		typeName = "double"
		initialValue = "0.0";
	case reflect.String:
		typeName = "String"
		initialValue = "\"\"";
	}
	return
}

func TypeToVarList(typeName string, obj KvoObject, types *[]KvoType) error {
	kt := KvoType{Name: typeName, Properties: make([]KvoProperty, 0, len(obj))}
	for k, t := range obj {
		if k == "__name__" {
			continue
		}
		v := reflect.ValueOf(t)
		typeName, initialValue, isComplex := TypeNameFromKind(v.Kind(), typeName, k, t)
		if typeName == "" {
			return ValidationError{
			Problem: "Don't know how to generate for value of kind " + v.Kind().String(),
			Field: typeName + "_" + k}
		}
		if isComplex {
			if err := TypeToVarList(typeName, t.(map[string]interface{}), types); err != nil {
				return err
			}
		}
		kt.Properties = append(kt.Properties, KvoProperty{Key: k, Type: typeName, InitialValue: initialValue})
	}
	*types = append(*types, kt)
	return nil
}

func Generate(javaPkg string, o map[string]interface{}, w io.Writer) (error) {
	if javaPkg == "" {
		return ValidationError{Problem: "Must specify a Java package.", Field: ""}
	}
	params := KvoTemplateParams{Package: javaPkg, Types: make([]KvoType, 0)}
	typeName, _, _ := TypeNameFromKind(reflect.ValueOf(o).Kind(), "", "", o)
	if typeName == "_" {
		return ValidationError{Problem: "Outermost object must specify a type name with key __name__", Field: ""}
	}
	if err := TypeToVarList(typeName, o, &params.Types); err != nil {
		return err
	}
	fmt.Println(params)

	if err := kvoTemplate.Execute(w, params); err != nil {
 		return err
	}
	return nil
}
