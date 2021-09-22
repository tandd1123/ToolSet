package parser

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xlab/treeprint"
	"gopkg.in/yaml.v2"
)

const (
	CONF_ROOT_DIR            = "cmd/conf"
	ANNOTATION_CMD_TYPE      = "cmd_type"
	ANNOTATION_CMD_TYPE_DIR  = "dir"
	ANNOTATION_CMD_TYPE_BIN  = "bin"
	ANNOTATION_CMD_TYPE_UTIL = "bin_util"
	ROOT_CMD_NAME_TOOLSET    = "toolset"
)

type CommandMap struct {
	cmdMap map[string]*cobra.Command
}

type CmdConfYamlFile struct {
	path     string
	info     os.FileInfo
	dirs     []string
	rootDir  string
	fileName string
	content  string
}

var commandMap *CommandMap

func (c *CommandMap) GetCommandByPath(path string) (*cobra.Command, bool) {
	//优先从子命令中查找
	cmd, ok := c.cmdMap[path]
	if ok {
		return cmd, true
	}
	return nil, false

}

func init() {
	commandMap = &CommandMap{
		cmdMap: make(map[string]*cobra.Command),
	}
}

func iterEmbedFsFiles(fs *embed.FS, dir string, filesMap map[string]os.FileInfo) {
	entrys, err := fs.ReadDir(dir)
	if err != nil {
		panic(fmt.Sprintf("embedFs err: %v", err))
	}
	for _, entry := range entrys {
		if entry.IsDir() {
			// fmt.Printf("dir: %v\n", entry.Name())
			iterEmbedFsFiles(fs, dir+"/"+entry.Name(), filesMap)
			continue
		}
		filesMap[dir+"/"+entry.Name()], _ = entry.Info()
		// fmt.Printf("obj: %v\n", entry.Name())
	}
}

func ParseCmd(fs embed.FS, rootCmd *cobra.Command) error {
	var files []*CmdConfYamlFile
	fs.ReadDir("conf")
	filesMap := make(map[string]os.FileInfo)
	iterEmbedFsFiles(&fs, "cmd/conf", filesMap)
	for path, info := range filesMap {
		// 仅当是文件 且 文件后缀是 yaml 时，添加到 files 数组
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".yaml") {
			buf, err := fs.ReadFile(path)
			if err != nil {
				panic(fmt.Errorf("read file err path: %v", path))
			}
			var dirs []string
			items := strings.Split(strings.TrimPrefix(path, CONF_ROOT_DIR+"/"), "/")
			if len(items) < 2 {
				panic(fmt.Errorf("invalid file path: %v", path))
			}
			for idx := range items {
				if idx < len(items)-1 {
					dirs = append(dirs, items[idx])
				}
			}

			file := &CmdConfYamlFile{
				path:     path,
				info:     info,
				rootDir:  dirs[0],
				fileName: info.Name(),
				dirs:     dirs,
				content:  string(buf),
			}
			files = append(files, file)
		}
	}

	for _, file := range files {
		// fmt.Printf("file: %+v", file)
		//初始化 yaml 文件路径上的 dir 路径信息
		if err := addDirCommand(rootCmd, file.dirs); err != nil {
			panic(fmt.Errorf("add dir command err: %v", err))
		}
		//添加具体的 yaml command
		if err := addYamlCommand(file.dirs, file.content); err != nil {
			panic(fmt.Errorf("add dir command err: %v", err))
		}
	}
	tree := treeprint.New()
	outputCommand(tree, rootCmd)
	fmt.Println(tree.String())
	return nil
}

func outputCommand(tree treeprint.Tree, cmd *cobra.Command) {
	//兜底， 在 cmd 为空的情况直接返回
	if cmd == nil {
		return
	}

	//叶子结点的命令则直接返回, 在本例子中 叶子结点就是 bin-util, 此处不输出。
	if len(cmd.Commands()) == 0 {
		// tree.AddNode(cmd.Name())
		return
	}

	//如果命令 cmd 对应的节点是 bin 类型, 则输出 usage.
	name := cmd.Name()

	//cmd help 用于命令对应的 help 执行命令
	cmdHelp := ""
	cmdType, ok := getAnnotionsCmdType(cmd)
	if ok && cmdType == ANNOTATION_CMD_TYPE_BIN {
		name = cmd.Use
		cmdHelp = cmd.Name() + " util: " + getCommandPath(cmd) + " help"
	}

	branch := tree
	// if name != ROOT_CMD_NAME_TOOLSET {
	branch = tree.AddBranch(name)
	// }
	if len(cmdHelp) != 0 {
		tree.AddNode(cmdHelp)
	}
	for _, subCmd := range cmd.Commands() {
		outputCommand(branch, subCmd)
	}
}

func getCommandPath(cmd *cobra.Command) string {
	var arr []string
	// tmpCmd := cmd
	for i := 0; i < 10; i++ {
		if cmd.Name() != ROOT_CMD_NAME_TOOLSET {
			arr = append([]string{cmd.Name()}, arr...)
		}
		if cmd.Parent() == nil {
			break
		}
		cmd = cmd.Parent()
	}
	// arr = append(arr, "help")
	arr = append([]string{os.Args[0]}, arr...)
	return strings.Join(arr, " ")
}

//新增 dir 类型的 命令
func addDirCommand(rootCmd *cobra.Command, dirs []string) error {
	for idx := range dirs {
		//检查 该 dir cmd 是否已添加过了。
		path := strings.Join(dirs[:idx+1], "/")
		cmd, ok := commandMap.GetCommandByPath(path)
		if !ok {
			cmd = newDirCommand(path, dirs[idx])
			commandMap.cmdMap[path] = cmd
		}

		//检查 该 dir cmd 是否已添加到上级 com 中
		parentCmdPath := ""
		parentCmd := rootCmd
		if idx-1 >= 0 {
			parentCmdPath = strings.Join(dirs[:idx], "/")
			parentCmd, _ = commandMap.GetCommandByPath(parentCmdPath)
		}
		if !containSubCommand(parentCmd, cmd) {
			parentCmd.AddCommand(cmd)
			// parentCmd.SuggestFor = append(parentCmd.SuggestFor, cmd.Name())
		}
	}
	return nil
}

func containSubCommand(currCmd *cobra.Command, subCmd *cobra.Command) bool {
	if len(currCmd.Commands()) == 0 {
		return false
	}
	for _, cmd := range currCmd.Commands() {
		if subCmd.Name() == cmd.Name() {
			return true
		}
	}
	return false
}

//新增 yaml 类型的 命令
func addYamlCommand(dirs []string, content string) error {
	//获取 yaml 命令的上一级目录 对应的 command
	parentCmd, _ := commandMap.GetCommandByPath(strings.Join(dirs, "/"))
	cmdv1 := &CmdV1{}
	if err := yaml.Unmarshal([]byte(content), &cmdv1); err != nil {
		panic(fmt.Errorf("cmd v1 yaml path: %v parse err: %v", strings.Join(dirs, "/"), err))
	}
	cmd := newYamlCommand(cmdv1, parentCmd)
	if !containSubCommand(parentCmd, cmd) {
		parentCmd.AddCommand(cmd)
		parentCmd.SuggestFor = append(parentCmd.SuggestFor, cmd.Name())
	}
	return nil
}

func newDirCommand(path, dir string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   fmt.Sprintf("%v [%v]", dir, path),
		Short: "for dir padding",
		Long:  "for dir padding",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("only for path padding: %v, sub command: %v\n", cmd.Use, strings.Join(cmd.SuggestFor, ","))
		},
	}
	setAnnotionsCmdType(cmd, ANNOTATION_CMD_TYPE_DIR)
	return cmd

}

func newYamlCommand(cmdv1 *CmdV1, parentCmd *cobra.Command) *cobra.Command {
	//1. 生成 install 和 doc 相关的命令信息
	var buf strings.Builder
	buf.WriteString("utils:\n")
	for idx := range cmdv1.Util {
		util := cmdv1.Util[idx]
		buf.WriteString(fmt.Sprintf("%v. %v: %v [%v]\n", idx, util.Name, util.Desc, getCommandPath(parentCmd)+" "+cmdv1.Cmd+" "+util.Name))
	}

	buf.WriteString("\ninstall:\n")
	for idx := range cmdv1.Install {
		install := cmdv1.Install[idx]
		buf.WriteString(fmt.Sprintf("os: %v, install cmd: %v\n", install.Name, install.InstallCmd))
	}

	buf.WriteString("\ndocs:\n")
	for idx := range cmdv1.Docs {
		doc := cmdv1.Docs[idx]
		buf.WriteString(fmt.Sprintf("idx: %v, doc: %v\n", idx, doc))
	}
	buf.WriteString("\n")

	//2. 构建命令
	utilCmd := &cobra.Command{
		Use:   fmt.Sprintf("%v [%v]", cmdv1.Cmd, cmdv1.Desc),
		Short: cmdv1.Desc,
		Long:  fmt.Sprintf("%v", buf.String()),
	}
	setAnnotionsCmdType(utilCmd, ANNOTATION_CMD_TYPE_BIN)

	//3. 构建可用于直接执行的简化的命令版本
	for idx := range cmdv1.Util {
		util := cmdv1.Util[idx]
		subCmd := &cobra.Command{
			Use:   fmt.Sprintf("%v [%v]", util.Name, util.Desc),
			Short: util.Args,
			Long:  util.Desc,
			Args:  cobra.NoArgs,
			Run: func(cmd *cobra.Command, args []string) {
				str := fmt.Sprintf("%v %v", utilCmd.Name(), cmd.Short)
				fmt.Printf("exec cmd detail: %v\n", str)
				execShellCmd(str)
			},
		}
		setAnnotionsCmdType(subCmd, ANNOTATION_CMD_TYPE_UTIL)
		utilCmd.AddCommand(subCmd)
	}
	return utilCmd
}

func execShellCmd(cmd string) error {
	c := exec.Command("/bin/bash", "-c", cmd)
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Env = os.Environ()
	// fmt.Printf("env: %v\n", strings.Join(c.Env, ","))
	err := c.Run()
	if err != nil {
		fmt.Printf("exec cmd: %v error %s\n", cmd, err)
		return err
	}
	return nil
}

func setAnnotionsCmdType(cmd *cobra.Command, cmdType string) {
	if cmd.Annotations == nil {
		cmd.Annotations = make(map[string]string)
	}
	cmd.Annotations[ANNOTATION_CMD_TYPE] = cmdType
}

func getAnnotionsCmdType(cmd *cobra.Command) (string, bool) {
	if cmd.Annotations == nil {
		return "", false
	}
	for k, v := range cmd.Annotations {
		if k == ANNOTATION_CMD_TYPE {
			return v, true
		}
	}
	return "", false
}
