package cmd

import (
	"fmt"
	"strings"

	comp "github.com/hbagdi/hit/pkg/completion"
	executorPkg "github.com/hbagdi/hit/pkg/executor"
)

func executeCompletion() error {
	fmt.Print(comp.Script)
	return nil
}

func completion() error {
	executor, err := executorPkg.NewExecutor(nil)
	if err != nil {
		return fmt.Errorf("initialize executor: %v", err)
	}
	defer executor.Close()
	err = executor.LoadFiles()
	if err != nil {
		return fmt.Errorf("read hit files: %v", err)
	}
	ids, err := executor.AllRequestIDs()
	if err != nil {
		return err
	}
	output := strings.Join(ids, " ")
	fmt.Println(output)
	return nil
}
