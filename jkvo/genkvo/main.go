package main;

import(
	"flag"
	"fmt"
//	"github.com/areusch/jkvo/jkvo"
	"jkvo"
	"os"
)

var outerClass = flag.String("outer_class", "", "The name of the containing class.")
var javaPkg = flag.String("java_package", "", "The Java package name.")

func main() {
	flag.Parse()

	if *outerClass == "" && *javaPkg == "" {
		flag.Usage()
		return
	}

	var spec jkvo.KvoObject
	var err error
	spec, err = jkvo.ParseAndValidate(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	err = jkvo.Generate(*javaPkg, *outerClass, spec, os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
}
