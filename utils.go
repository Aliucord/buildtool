package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func handleErr(err error) {
	if err != nil {
		fatal(err)
	}
}

func fatal(args ...interface{}) {
	colorPrint(ERROR, args...)
	os.Exit(1)
}

func colorPrint(color string, args ...interface{}) {
	fmt.Print(color)
	fmt.Print(args...)
	fmt.Println(RESET)
}

func gradlew(stdout io.Writer, dir string, args ...string) {
	if runtime.GOOS == "windows" {
		execCmd(stdout, dir, "cmd", "/k", "gradlew.bat "+strings.Join(args, " ")+" && exit")
	} else {
		execCmd(stdout, dir, "./gradlew", args...)
	}
}

func execCmd(stdout io.Writer, dir string, c string, args ...string) {
	cmd := exec.Command(c, args...)
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	cmd.Stdout = stdout
	handleErr(cmd.Run())
}

func writePluginEntry(zipw *zip.Writer, pluginName string) {
	zipFilew, _ := zipw.Create("ac-plugin")
	zipFilew.Write([]byte(pluginName))
}
