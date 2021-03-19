# komando

Build beautiful CLI experiences out of the box in Go.

Komando provides simple terminal UI components like `Header`, `StepGroup`, `NamedValue` and more to help you craft great informative CLI experiences quickly.  

Stop wasting time customising lower level components to build what you need.

All components are easily testable with a Fake UI implementation and customisable using intuitive overrides.

## Quickstart

```go
package main

import "github.com/appvia/komando"

func main() {
	ui := komando.ConsoleUI()
	ui.Header("Initialising project...") 
	ui.Output("Running init framework using available steps")

	sg := ui.StepGroup()
	defer sg.Done()

	step1 := sg.Add("Step 1 - validate")
	step2 := sg.Add("Step 2 - execute")
	step1.Success()
	step2.Error()

	ui.Output("")
	ui.Output("Initialisation has failed!\nhere's what do next...", komando.WithErrorBoldStyle())
}
```

On terminal
```shell
» Initialising project...
Running init framework using available steps
 ✓ Step 1 - validate   # foreground color is green
 ✕ Step 2 - execute    # foreground color is red
 
✕ Initialisation has failed! # Bold with foreground color red
  here's what do next...     # No foreground color
```
