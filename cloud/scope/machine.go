/*
Copyright 2023 Teppei Sudo.

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

package scope

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sp-yduck/proxmox/pkg/service/node/vm"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/controllers/noderefutil"
	capierrors "sigs.k8s.io/cluster-api/errors"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/sp-yduck/cluster-api-provider-proxmox/api/v1beta1"
	"github.com/sp-yduck/cluster-api-provider-proxmox/cloud/providerid"
	"github.com/sp-yduck/proxmox/pkg/service"
)

type MachineScopeParams struct {
	ProxmoxServices
	Client         client.Client
	Machine        *clusterv1.Machine
	ProxmoxMachine *infrav1.ProxmoxMachine
	ClusterGetter  *ClusterScope
}

func NewMachineScope(params MachineScopeParams) (*MachineScope, error) {
	if params.Client == nil {
		return nil, errors.New("client is required when creating a MachineScope")
	}
	if params.Machine == nil {
		return nil, errors.New("failed to generate new scope from nil Machine")
	}
	if params.ProxmoxMachine == nil {
		return nil, errors.New("failed to generate new scope from nil ProxmoxMachine")
	}

	helper, err := patch.NewHelper(params.ProxmoxMachine, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}

	return &MachineScope{
		client:         params.Client,
		Machine:        params.Machine,
		ProxmoxMachine: params.ProxmoxMachine,
		patchHelper:    helper,
		ClusterGetter:  params.ClusterGetter,
	}, err
}

type MachineScope struct {
	client         client.Client
	patchHelper    *patch.Helper
	Machine        *clusterv1.Machine
	ProxmoxMachine *infrav1.ProxmoxMachine
	ClusterGetter  *ClusterScope
}

func (m *MachineScope) CloudClient() *service.Service {
	return m.ClusterGetter.CloudClient()
}

func (m *MachineScope) Name() string {
	return m.ProxmoxMachine.Name
}

func (m *MachineScope) Namespace() string {
	return m.ProxmoxMachine.Namespace
}

// func (m *MachineScope) Client() Compute {
// 	return m.ClusterGetter.Client()
// }

func (m *MachineScope) Close() error {
	return m.PatchObject()
}

func (m *MachineScope) GetInstanceStatus() *infrav1.InstanceStatus {
	return m.ProxmoxMachine.Status.InstanceStatus
}

// SetInstanceStatus sets the ProxmoxMachine instance status.
func (m *MachineScope) SetInstanceStatus(v infrav1.InstanceStatus) {
	m.ProxmoxMachine.Status.InstanceStatus = &v
}

func (m *MachineScope) GetInstanceID() *string {

	parsed, err := noderefutil.NewProviderID(m.GetProviderID())
	if err != nil {
		return nil
	}
	return pointer.StringPtr(parsed.ID())
}

func (m *MachineScope) GetProviderID() string {
	if m.ProxmoxMachine.Spec.ProviderID != nil {
		return *m.ProxmoxMachine.Spec.ProviderID
	}
	return ""
}

// SetProviderID sets the ProxmoxMachine providerID in spec.
func (m *MachineScope) SetProviderID(instance *vm.VirtualMachine) error {
	providerid, err := providerid.New(instance.Node, instance.VMID)
	if err != nil {
		return err
	}
	m.ProxmoxMachine.Spec.ProviderID = pointer.StringPtr(providerid.String())
	return nil
}

func (m *MachineScope) SetReady() {
	m.ProxmoxMachine.Status.Ready = true
}

func (m *MachineScope) SetFailureMessage(v error) {
	m.ProxmoxMachine.Status.FailureMessage = pointer.StringPtr(v.Error())
}

func (m *MachineScope) SetFailureReason(v capierrors.MachineStatusError) {
	m.ProxmoxMachine.Status.FailureReason = &v
}

// PatchObject persists the cluster configuration and status.
func (s *MachineScope) PatchObject() error {
	return s.patchHelper.Patch(context.TODO(), s.ProxmoxMachine)
}