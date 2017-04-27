/*
Copyright (c) 2016 VMware, Inc. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package vm

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path"

	"github.com/vmware/govmomi/govc/cli"
	"github.com/vmware/govmomi/govc/flags"
	"github.com/vmware/govmomi/govc/importx"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/progress"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
)

type export struct {
	*flags.ClientFlag
	*flags.VirtualMachineFlag

	priority types.VirtualMachineMovePriority
	spec     types.VirtualMachineRelocateSpec
}

func init() {
	cli.Register("vm.export", &export{})
}

func (cmd *export) Register(ctx context.Context, f *flag.FlagSet) {

	cmd.ClientFlag, ctx = flags.NewClientFlag(ctx)
	cmd.ClientFlag.Register(ctx, f)

	cmd.VirtualMachineFlag, ctx = flags.NewVirtualMachineFlag(ctx)
	cmd.VirtualMachineFlag.Register(ctx, f)

}

func (cmd *export) Process(ctx context.Context) error {
	if err := cmd.ClientFlag.Process(ctx); err != nil {
		return err
	}
	if err := cmd.VirtualMachineFlag.Process(ctx); err != nil {
		return err
	}

	return nil
}

func (cmd *export) Usage() string {
	return "DEST"
}

func (cmd *export) Description() string {
	return `Exports VM to DEST on the local system.

Examples:
  govc vm.export -vm=my-vm  my-dirname`
}

func (cmd *export) Run(ctx context.Context, f *flag.FlagSet) error {
	//c, err := cmd.Client()
	//if err != nil {
	//	return err
	//}
	//fmt.Printf("client: %v\n", c)

	vm, err := cmd.VirtualMachine()
	if err != nil {
		return err
	}

	if vm == nil {
		return errors.New("No VM specified")
	}

	lease, err := vm.ExportVm(ctx)
	if err != nil {
		return err
	}

	info, err := lease.Wait(ctx)
	if err != nil {
		return err
	}

	var items []importx.OvfFileItem

	for _, device := range info.DeviceUrl {
		file_size := device.FileSize
		target_id := device.TargetId
		//for _, item := range vm.DiskId {
		if target_id == "" {
			fmt.Printf("skipping device %v\n", device)
			continue
		}
		url, err := vm.Client().ParseURL(device.Url)
		if err != nil {
			return err
		}

		//	u, err := vm.Client().ParseURL(device.Url)
		//	if err != nil {
		//		return err
		//	}

		//i := importx.OvfFileItem{
		//url:  u,
		//item: item,
		//ch:   make(chan progress.Report),
		//}

		//items = append(items, i)
		//}
		fmt.Printf("device target:%s\n", target_id)
		fmt.Printf("file size :%d\n", file_size)
		fmt.Printf("url :%s\n", url)
	}

	u := importx.NewLeaseUpdater(vm.Client(), lease, items)
	defer u.Done()

	//for _, i := range items {
	//	err = cmd.download(lease, i)
	//	if err != nil {
	//		return err
	//	}
	//}

	return lease.HttpNfcLeaseComplete(ctx)
}

func (cmd *export) download(lease *object.HttpNfcLease, ofi importx.OvfFileItem) error {
	item := ofi.Item
	file := item.Path

	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	logger := cmd.ProgressLogger(fmt.Sprintf("Downloading %s... ", path.Base(file)))
	defer logger.Wait()

	opts := soap.Download{
		//ContentLength: size,
		Progress: progress.Tee(ofi, logger),
	}

	// Non-disk files (such as .iso) use the PUT method.
	// Overwrite: t header is also required in this case (ovftool does the same)
	if item.Create {
		opts.Method = "PUT"
		opts.Headers = map[string]string{
			"Overwrite": "t",
		}
	} else {
		opts.Method = "POST"
		opts.Type = "application/x-vnd.vmware-streamVmdk"
	}

	//c := cmd.Client
	return c.DownloadFile("test", ofi.Url, &opts)
	//func (c *Client) DownloadFile(file string, u *url.URL, param *Download) error {
	//return nil
}
