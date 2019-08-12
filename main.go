package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "my string representation"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var myFlags arrayFlags

func CalcHash(s string) string { //get hash of folder and it's contetn
	out, _ := exec.Command(
		"sh",
		"-c",
		fmt.Sprintf("find %s -type f -print0 | sort -z | xargs -0 shasum | shasum",
			s),
	).Output()
	return string(out)
}
func main() {

	flag.Var(&myFlags, "watch", "src of folder to be whatched")
	runner := flag.String("runner", "go run main.go", "your command to run ")
	flag.Parse()

	SIGNALS := make(chan os.Signal, 1)
	DONE := make(chan bool, 1)
	signal.Notify(SIGNALS, syscall.SIGTERM, syscall.SIGINT)

	Output := CalcHash(myFlags[0])

	cmd := exec.Command("sh", "-c", *runner)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	go func(init *string) {
		fmt.Println("\033[32m** Start Hot Reloading \033[0m")
		for {
			Output := CalcHash(myFlags[0])
			if *init != Output {
				*init = Output

				pgid, err := syscall.Getpgid(cmd.Process.Pid)
				if err == nil {
					fmt.Println("killing ", pgid)
					syscall.Kill(-pgid, 15) // note the minus sign
				} else {
					fmt.Println("err cause ", err)
				}

				fmt.Println(
					fmt.Sprintf("\033[31mkill process[%d] with parent[%d] again\n\033[33mreloadin...\033[0m",
						cmd.Process.Pid,
						os.Getpid(),
					),
				)
				cmd = exec.Command("sh", "-c", *runner)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Stdin = os.Stdin

				cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

				if err := cmd.Start(); err != nil {
					log.Fatalf("Start(runner) failed with %s\n", err)
				}
			}
			time.Sleep(time.Second)
		}
		fmt.Println("for finished")
	}(&Output)

	if err := cmd.Start(); err != nil {
		log.Fatalf("Start(runner) failed with %s\n", err)
		DONE <- true
	}
	go func() {
		<-SIGNALS
		DONE <- true
		pgid, _ := syscall.Getpgid(cmd.Process.Pid)
		syscall.Kill(-pgid, 15)
		fmt.Println("\n\033[34mFinishing Hot Reloading\nBye\033[0m")
	}()

	<-DONE

}
