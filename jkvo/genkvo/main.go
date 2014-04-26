package main;

import(
	"flag"
	"fmt"
	"github.com/areusch/jkvo/jkvo"
	"os"
)

var javaPkg = flag.String("java_package", "", "The Java package name.")

func main() {
	flag.Parse()

	var spec jkvo.KvoObject
	var err error
	spec, err = jkvo.ParseAndValidate(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	err = jkvo.Generate(*javaPkg, spec, os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
}
