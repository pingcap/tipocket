package nemesis

import (
	"context"
	"fmt"
	"math/rand"
	"reflect"
	"time"

	"github.com/juju/errors"
	"github.com/pingcap/pd/v4/tools/pd-ctl/pdctl/command"

	"github.com/spf13/cobra"

	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/core"
)

const (
	allFlag  = 0b111
	tidbFlag = 0b001
	pdFlag   = 0b010
	tikvFlag = 0b100
)

type dynamicConfigGenerator struct {
	name string
}

// Generate generates container-kill actions, to simulate the case that node can be recovered quickly after being killed
func (g dynamicConfigGenerator) Generate(nodes []clusterTypes.Node) []*core.NemesisOperation {
	var flag int
	nodes = findPDMember(nodes, true)

	switch g.name {
	case "all_dynamic_config":
		flag = allFlag
	case "tidb_dynamic_config":
		flag = tidbFlag
	case "pd_dynamic_config":
		flag = pdFlag
	case "tikv_dynamic_config":
		flag = tikvFlag
	}

	return dynamicConfigNemesis(nodes, flag)
}

// Name of the nemesis
func (g dynamicConfigGenerator) Name() string {
	return g.name
}

// NewDynamicConfigGenerator creates a generator.
// Name is all.
func NewDynamicConfigGenerator(name string) core.NemesisGenerator {
	return dynamicConfigGenerator{name: name}
}

func dynamicConfigNemesis(nodes []clusterTypes.Node, flag int) []*core.NemesisOperation {
	var ops []*core.NemesisOperation

	if flag > len(nodes) {
		flag = len(nodes)
	}

	if op := tidbDynamicConfigNemesis(nodes, flag); op != nil {
		ops = append(ops, op)
	}
	if op := pdDynamicConfigNemesis(nodes, flag); op != nil {
		ops = append(ops, op)
	}
	if op := tikvDynamicConfigNemesis(nodes, flag); op != nil {
		ops = append(ops, op)
	}

	return ops
}

func tidbDynamicConfigNemesis(nodes []clusterTypes.Node, flag int) *core.NemesisOperation {
	if flag&tidbFlag == 0 {
		return nil
	}
	freq := time.Second * time.Duration(rand.Intn(120)+60)

	return &core.NemesisOperation{
		Type: core.DynamicConfig,
		Node: &nodes[0],
		// pass nodes in args here
		// maybe handle pd-ctl error in the future
		// if this error is caused by a pd pod kill
		// we can retry on another pd node
		InvokeArgs:  []interface{}{"tidb", nodes},
		RecoverArgs: []interface{}{"tidb", nodes},
		RunTime:     freq,
	}
}

func pdDynamicConfigNemesis(nodes []clusterTypes.Node, flag int) *core.NemesisOperation {
	if flag&tidbFlag == 0 {
		return nil
	}
	freq := time.Second * time.Duration(rand.Intn(120)+60)

	return &core.NemesisOperation{
		Type:        core.DynamicConfig,
		Node:        &nodes[0],
		InvokeArgs:  []interface{}{"pd", nodes},
		RecoverArgs: []interface{}{"pd", nodes},
		RunTime:     freq,
	}
}

func tikvDynamicConfigNemesis(nodes []clusterTypes.Node, flag int) *core.NemesisOperation {
	if flag&tidbFlag == 0 {
		return nil
	}
	freq := time.Second * time.Duration(rand.Intn(120)+60)

	return &core.NemesisOperation{
		Type:        core.DynamicConfig,
		Node:        &nodes[0],
		InvokeArgs:  []interface{}{"tikv", nodes},
		RecoverArgs: []interface{}{"tikv", nodes},
		RunTime:     freq,
	}
}

// containerKill implements Nemesis
type dynamicConfig struct {
	boforeStates map[string]string
}

// Invoke dynamic nemesis
func (dynamicConfig) Invoke(ctx context.Context, node *clusterTypes.Node, args ...interface{}) error {
	if len(args) != 2 {
		return errors.New("length of args is not 2")
	}
	component, ok := args[0].(string)
	if !ok {
		return errors.Errorf("args[0] not string, got %s", reflect.TypeOf(args[0]))
	}
	nodes, ok := args[1].([]clusterTypes.Node)
	if !ok {
		return errors.Errorf("args[0] not []clusterTypes.Node, got %s", reflect.TypeOf(args[1]))
	}

	if len(nodes) != 0 {
		node = &nodes[0]
	}

	cfgKey, cfgVal := randDynamicConfig(component)

	commands := []string{
		"-u",
		fmt.Sprintf("http://%s:%d", node.IP, node.Port),
		"component",
		"set",
		component,
		cfgKey,
		cfgVal,
	}

	return errors.Trace(executePDCtl(commands))
}

// Recover dynamic nemesis
func (dynamicConfig) Recover(ctx context.Context, node *clusterTypes.Node, args ...interface{}) error {
	return errors.Trace(executePDCtl([]string{"", "", ""}))
}

func (dynamicConfig) Name() string {
	return string(core.DynamicConfig)
}

func randDynamicConfig(configType string) (string, string) {
	var dynamicConfigItemsPtr *[]dynamicConfigItem
	switch configType {
	case "tidb":
		dynamicConfigItemsPtr = &tidbDynamicConfigItems
	case "tikv":
		dynamicConfigItemsPtr = &tikvDynamicConfigItems
	case "pd":
		dynamicConfigItemsPtr = &pdDynamicConfigItems
	}

	dynamicConfigItems := *dynamicConfigItemsPtr
	if len(dynamicConfigItems) == 0 {
		return "", ""
	}

	item := dynamicConfigItems[rand.Intn(len(dynamicConfigItems))]
	return item.key, item.rand()
}

// The following code is ported from PD ctl

// CommandFlags are flags that used in all Commands
type CommandFlags struct {
	URL      string
	CAPath   string
	CertPath string
	KeyPath  string
	Help     bool
}

func getBasicCmd() (*cobra.Command, *CommandFlags) {
	commandFlags := CommandFlags{}
	rootCmd := &cobra.Command{
		Use:   "pd-ctl",
		Short: "Placement Driver control",
	}

	rootCmd.PersistentFlags().StringVarP(&commandFlags.URL, "pd", "u", "http://127.0.0.1:2379", "Address of pd.")
	rootCmd.Flags().StringVar(&commandFlags.CAPath, "cacert", "", "Path of file that contains list of trusted SSL CAs.")
	rootCmd.Flags().StringVar(&commandFlags.CertPath, "cert", "", "Path of file that contains X509 certificate in PEM format.")
	rootCmd.Flags().StringVar(&commandFlags.KeyPath, "key", "", "Path of file that contains X509 key in PEM format.")
	rootCmd.PersistentFlags().BoolVarP(&commandFlags.Help, "help", "h", false, "Help message.")

	rootCmd.AddCommand(
		command.NewConfigCommand(),
		command.NewRegionCommand(),
		command.NewStoreCommand(),
		command.NewStoresCommand(),
		command.NewMemberCommand(),
		command.NewExitCommand(),
		command.NewLabelCommand(),
		command.NewPingCommand(),
		command.NewOperatorCommand(),
		command.NewSchedulerCommand(),
		command.NewTSOCommand(),
		command.NewHotSpotCommand(),
		command.NewClusterCommand(),
		command.NewHealthCommand(),
		command.NewLogCommand(),
		command.NewPluginCommand(),
		command.NewComponentCommand(),
	)

	rootCmd.Flags().ParseErrorsWhitelist.UnknownFlags = true
	rootCmd.SilenceErrors = true

	return rootCmd, &commandFlags
}

func getMainCmd(args []string) (*cobra.Command, *CommandFlags) {
	rootCmd, commandFlags := getBasicCmd()

	// rootCmd.Run = pdctlRun

	rootCmd.SetArgs(args)
	rootCmd.ParseFlags(args)

	return rootCmd, commandFlags
}

func executePDCtl(cmds []string) error {
	rootCmd, commandFlags := getMainCmd(cmds)
	if len(commandFlags.CAPath) != 0 {
		if err := command.InitHTTPSClient(commandFlags.CAPath, commandFlags.CertPath, commandFlags.KeyPath); err != nil {
			return err
		}
	}

	return errors.Trace(rootCmd.Execute())
}
