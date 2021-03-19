# komando

Build beautiful CLI experiences out of the box in Go.

Komando provides simple terminal UI components like `Header`, `StepGroup`, `NamedValue` and more to help you craft great informative CLI experiences quickly.

Stop wasting time customising lower level components to build what you need.

All components are easily testable with a Fake UI implementation and customisable using intuitive overrides.

## Quickstart

```go
package main

import (
	"fmt"
	"time"

	komando "github.com/appvia/komando"
)

func main() {
	ui := komando.ConsoleUI()
	ui.Header("Initialising project...")
	ui.Output("Running init framework using available steps")

	sg := ui.StepGroup()
	defer sg.Done()

	step1 := sg.Add("Step 1 - validate")
	step2 := sg.Add("Step 2 - execute")

	step1.Success()
	step2.Warning()

	for i := 0; i < 5; i++ {
		ui.Output(
			fmt.Sprintf("Step [2] substep [%v] - sleep 0.5s", i),
			komando.WithStyle(komando.LogStyle),
			komando.WithIndentChar(komando.LogIndentChar),
			komando.WithIndent(3),
		)
		time.Sleep(500 * time.Millisecond)
	}

	step3 := sg.Add("Step 3 - finalize")
	step3.Error()

	ui.Output("")
	ui.Output("Initialisation has failed!\nhere's what to do next...", komando.WithErrorBoldStyle())
}
```

Output:

[![In terminal](./docs/assets/example.gif)](./docs/assets/example.gif)
