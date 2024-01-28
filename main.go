package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"slices"
	"syscall"
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
	fmt.Printf("- [\"%s\",%d,%d]\n", current_workspace, last, diff)
	if err := DBNotify(current_workspace, last, diff); err != nil {
		log.Println(err)
	}
	current_workspace = name
	last = now
}

func main() {
	OpenDB()

	last = time.Now().UnixMilli()

	// Get the current workspace to get going
	{
		wss, err := i3.GetWorkspaces()
		if err != nil {
			log.Fatal("can't get workspaces")
		}

		idx := slices.IndexFunc(wss, func(w i3.Workspace) bool { return w.Focused })
		if idx > -1 {
			current_workspace = re_num_replace.ReplaceAllString(wss[idx].Name, "")
		}

	}

	go pollIdle()

	// We still want to emit something if we were interrupted or terminated.
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		var sig = <-signalChan
		notify("")
		db.Close()
		if sig == syscall.SIGINT {
			os.Exit(130)
		}
		os.Exit(0)
	}()

	rec := i3.Subscribe("workspace", "window")
	for rec.Next() {
		evt := rec.Event()

		switch v := evt.(type) {

		case *i3.WorkspaceEvent:
			if v.Change == "focus" {
				notify(v.Current.Name)
			}

		case *i3.WindowEvent:
			// This is how we detect that a scratchpad window was put forth
			if v.Container.ScratchpadState != "none" {
				var now = time.Now().UnixMilli()
				var diff = now - last
				if diff > 20 {
					// Usually, the scratchpad event is called right after the workspace focus event when the window is hidden back to the scratchpad so we arbitrarily take it into account only if it's been more than 20 ms
					notify("__i3_scratch")
				}
			}

			// Special rule for VScode, but should probably be in a config file somewhere ; if a window matching this regexp is focused to, the workspace gets the name as number: project name
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
