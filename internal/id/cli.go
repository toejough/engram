package id

import "fmt"

// NextArgs holds arguments for the id next command.
type NextArgs struct {
	Type string `targ:"flag,short=t,required,desc=ID type (REQ / DES / ARCH / TASK / ISSUE)"`
	Dir  string `targ:"flag,short=d,default=.,desc=Project directory to scan"`
}

// RunNext generates the next ID of the given type.
func RunNext(args NextArgs) error {
	result, err := Next(args.Dir, args.Type)
	if err != nil {
		return err
	}

	fmt.Println(result)

	return nil
}
