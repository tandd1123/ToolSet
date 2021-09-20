package main

import (
	"embed"
	_ "embed"

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
	rootCmd.Execute()
}
