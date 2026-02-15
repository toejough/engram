package main

import (
	"fmt"

	"github.com/toejough/projctl/internal/id"
)

type idNextArgs struct {
	Type string `targ:"flag,short=t,required,desc=ID type (REQ / DES / ARCH / TASK / ISSUE)"`
	Dir  string `targ:"flag,short=d,default=.,desc=Project directory to scan"`
}

func idNext(args idNextArgs) error {
	result, err := id.Next(args.Dir, args.Type)
	if err != nil {
		return err
	}

	fmt.Println(result)
	return nil
}
