package main

import (
	"fmt"
	"log"
	"regexp"
	"slices"
	"time"

	"github.com/ka2n/go-idle"
	"go.i3wm.org/i3/v4"
)

///////////////////////////////////////////////////////////

var re_vscode = regexp.MustCompile(`.*? - (.*) - Visual Studio Code`)
var re_num = regexp.MustCompile(`^\d+$`)
var re_num_replace = regexp.MustCompile(`^\d+\s*:\s*`)

var IDLE_THRESHOLD int64 = 60000
var currently_idle = false
var idle_value int64 = -1

/*
Poll the operating system for idle time.
*/
func pollIdle() {
	var err error
	var idleTime time.Duration

	for err == nil {
		idleTime, err = idle.Get()
		milliseconds := idleTime.Milliseconds()

		if milliseconds > IDLE_THRESHOLD && !currently_idle {
			idle_value = idleTime.Milliseconds()
			currently_idle = true
			before_idle = current_workspace
			notify("__idle")
			// log.Printf("Idle for %d seconds.", int(idleTime.Seconds()))
		} else if milliseconds < idle_value {
			// We stopped being idle !
			idle_value = -1
			currently_idle = false
			if before_idle != "" {
				notify(before_idle)
				before_idle = ""
			}
		}

		time.Sleep(1000000000) // 1 second, 10^9 nanoseconds
	}

	if err != nil {
		log.Fatal(err)
	}

}

var before_idle = ""
var current_workspace = ""
var last int64 = -1

func notify(name string) {
	name = re_num_replace.ReplaceAllString(name, "")

	var now = time.Now().UnixMilli()
	var diff = now - last
	// pp.Println(name)
	fmt.Printf("- [\"%s\",%d]\n", current_workspace, diff)
	current_workspace = name
	last = now
}

func main() {
	last = time.Now().UnixMilli()
	fmt.Println("-", last)

	rec := i3.Subscribe("workspace", "window")
	wss, err := i3.GetWorkspaces()
	if err != nil {
		log.Fatal("can't get workspaces")
	}

	idx := slices.IndexFunc(wss, func(w i3.Workspace) bool { return w.Focused })
	if idx > -1 {
		current_workspace = re_num_replace.ReplaceAllString(wss[idx].Name, "")
	}

	go pollIdle()

	// i3.RunCommand()

	for rec.Next() {
		evt := rec.Event()
		// pp.Println(evt)

		switch v := evt.(type) {

		case *i3.WorkspaceEvent:
			// pp.Println(v.Change)
			if v.Change == "focus" {
				notify(v.Current.Name)
			}
			// pp.Println("Workspace: ", v.Current.Name)

		case *i3.WindowEvent:
			if v.Container.ScratchpadState != "none" {
				var now = time.Now().UnixMilli()
				var diff = now - last
				if diff > 20 {
					// Usually, the scratchpad event is called right after the workspace focus event
					// so we only take it into account if it's been more than 20 ms
					notify("__i3_scratch")
				}
			}

			if re_num.MatchString(current_workspace) {
				var title = v.Container.WindowProperties.Title
				var groups = re_vscode.FindStringSubmatch(title)
				if len(groups) > 0 {
					project := groups[1]
					var newname = fmt.Sprintf(`%s: %s`, current_workspace, project)
					notify(newname)
					i3.RunCommand(fmt.Sprintf(`rename workspace to "%s"`, newname))
				}
			}
		}
	}
}
