package cmd

import (
	"fmt"
	"strings"

	comp "github.com/hbagdi/hit/pkg/completion"
)

func executeCompletion() error {
	fmt.Print(comp.Script)
	return nil
}

func completion() error {
	ids, err := requestIDs()
	if err != nil {
		return err
	}
	output := strings.Join(ids, " ")
	fmt.Println(output)
	return nil
}
