package main

import (
	"embed"
	_ "embed"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tandd1123/ToolSet/cmd/parser"
)

//go:embed cmd/conf/*
var embedFs embed.FS

func main() {
	rootCmd := &cobra.Command{
		Use:  "toolset [tool set manager]",
		Args: cobra.NoArgs,
	}
	parser.ParseCmd(embedFs, rootCmd)
	if len(os.Args) == 1 {
		fmt.Printf("[info]: no detect args input, show toolset all support cmds below:\n")
		parser.OutputCommands(rootCmd)
		os.Exit(0)
	}
	rootCmd.Execute()
}
