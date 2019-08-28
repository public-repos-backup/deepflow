/*
 * rpc is a subcommands of droplet-ctl
 * which pulls policy information from controller by rpc.
 * now it contains 3 subcommands:
 *   flowAcls     get flowAcls from controller
 *   ipGroups     get ipGroups from controller
 *   platformData get platformData from controller
 */
package rpc

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sort"

	"github.com/spf13/cobra"
	"gitlab.x.lan/yunshan/droplet/config"
	"gitlab.x.lan/yunshan/droplet/dropletctl"
	"gitlab.x.lan/yunshan/message/trident"
)

type CmdExecute func(response *trident.SyncResponse)
type SortedAcls []*trident.FlowAcl

func regiterCommand() []*cobra.Command {
	platformDataCmd := &cobra.Command{
		Use:   "platformData",
		Short: "get platformData from controller, press Ctrl^c to end it",
		Run: func(cmd *cobra.Command, args []string) {
			initCmd(platformData)
		},
	}
	ipGroupsCmd := &cobra.Command{
		Use:   "ipGroups",
		Short: "get ipGroups from controller, press Ctrl^c to end it",
		Run: func(cmd *cobra.Command, args []string) {
			initCmd(ipGroups)
		},
	}
	flowAclsCmd := &cobra.Command{
		Use:   "flowAcls",
		Short: "get flowAcls from controller, press Ctrl^c to end it",
		Run: func(cmd *cobra.Command, args []string) {
			initCmd(flowAcls)
		},
	}

	commands := []*cobra.Command{platformDataCmd, ipGroupsCmd, flowAclsCmd}
	return commands
}

func RegisterRpcCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   "rpc",
		Short: "pull policy from controller by rpc",
	}
	cmds := regiterCommand()
	for _, handler := range cmds {
		root.AddCommand(handler)
	}

	return root
}

func initCmd(cmd CmdExecute) {
	cfg := config.Load(dropletctl.ConfigPath)

	controllers := make([]net.IP, 0, len(cfg.ControllerIps))
	for _, ipString := range cfg.ControllerIps {
		ip := net.ParseIP(ipString)
		controllers = append(controllers, ip)
	}

	synchronizer := config.NewRpcConfigSynchronizer(controllers, cfg.ControllerPort, cfg.RpcTimeout)
	synchronizer.Register(func(response *trident.SyncResponse, version *config.RpcInfoVersions) {
		cmd(response)
		fmt.Println("press Ctrl^c to end it !!")
	})

	synchronizer.Start()

	wait := make(chan os.Signal)
	signal.Notify(wait, os.Interrupt)
	if sig := <-wait; sig != os.Interrupt {
		fmt.Println("press Ctrl^c to end it !!")
	}
}

func JsonFormat(index int, v interface{}) {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		fmt.Println("json encode failed")
	}
	fmt.Printf("\t%v: %s\n", index, jsonBytes)
}

func (a SortedAcls) Len() int {
	return len(a)
}

func (a SortedAcls) Less(i, j int) bool {
	return a[i].GetId() < a[j].GetId()
}

func (a SortedAcls) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func flowAcls(response *trident.SyncResponse) {
	flowAcls := trident.FlowAcls{}
	fmt.Println("flow Acls version:", response.GetVersionAcls())

	if flowAclsCompressed := response.GetFlowAcls(); flowAclsCompressed != nil {
		if err := flowAcls.Unmarshal(flowAclsCompressed); err == nil {
			sort.Sort(SortedAcls(flowAcls.FlowAcl)) // sort by id
			fmt.Println("flow Acls:")
			for index, entry := range flowAcls.FlowAcl {
				JsonFormat(index+1, entry)
			}
		}
	}
}

func ipGroups(response *trident.SyncResponse) {
	groups := trident.Groups{}
	fmt.Println("Groups version:", response.GetVersionGroups())

	if groupsCompressed := response.GetGroups(); groupsCompressed != nil {
		if err := groups.Unmarshal(groupsCompressed); err == nil {
			fmt.Println("Groups data:")
			for index, entry := range groups.Groups {
				JsonFormat(index+1, entry)
			}
		}
	}
}

func platformData(response *trident.SyncResponse) {
	platform := trident.PlatformData{}
	fmt.Println("PlatformData version:", response.GetVersionPlatformData())

	if plarformCompressed := response.GetPlatformData(); plarformCompressed != nil {
		if err := platform.Unmarshal(plarformCompressed); err == nil {
			fmt.Println("interfaces:")
			for index, entry := range platform.Interfaces {
				JsonFormat(index+1, entry)
			}
			fmt.Println("peer connections:")
			for index, entry := range platform.PeerConnections {
				JsonFormat(index+1, entry)
			}
		}
	}
}
