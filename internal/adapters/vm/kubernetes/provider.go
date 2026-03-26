package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	kexec "github.com/popododo0720/anarchy/internal/adapters/kubernetes/exec"
	domainvm "github.com/popododo0720/anarchy/internal/domain/vm"
)

type Provider struct {
	runner    kexec.Runner
	namespace string
}

func NewProvider(runner kexec.Runner, namespace string) Provider {
	if namespace == "" {
		namespace = "anarchy-system"
	}
	return Provider{runner: runner, namespace: namespace}
}

type vmListResponse struct {
	Items []struct {
		Metadata struct {
			Name string `json:"name"`
		} `json:"metadata"`
		Spec struct {
			Template struct {
				Spec struct {
					Domain struct {
						CPU struct {
							Cores int `json:"cores"`
						} `json:"cpu"`
						Resources struct {
							Requests struct {
								Memory string `json:"memory"`
							} `json:"requests"`
						} `json:"resources"`
					} `json:"domain"`
					Networks []struct {
						Name string `json:"name"`
					} `json:"networks"`
				} `json:"spec"`
				Metadata struct {
					Annotations map[string]string `json:"annotations"`
				} `json:"metadata"`
			} `json:"template"`
		} `json:"spec"`
		Status struct {
			PrintableStatus string `json:"printableStatus"`
		} `json:"status"`
	} `json:"items"`
}

type vmSingleResponse struct {
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Spec struct {
		Template struct {
			Spec struct {
				Domain struct {
					CPU struct {
						Cores int `json:"cores"`
					} `json:"cpu"`
					Resources struct {
						Requests struct {
							Memory string `json:"memory"`
						} `json:"requests"`
					} `json:"resources"`
				} `json:"domain"`
				Networks []struct {
					Name string `json:"name"`
				} `json:"networks"`
			} `json:"spec"`
			Metadata struct {
				Annotations map[string]string `json:"annotations"`
			} `json:"metadata"`
		} `json:"template"`
	} `json:"spec"`
	Status struct {
		PrintableStatus string `json:"printableStatus"`
	} `json:"status"`
}

type vmiListResponse struct {
	Items []struct {
		Metadata struct {
			Name string `json:"name"`
		} `json:"metadata"`
		Status struct {
			Interfaces []struct {
				IPAddress string `json:"ipAddress"`
			} `json:"interfaces"`
		} `json:"status"`
	} `json:"items"`
}

type vmiSingleResponse struct {
	Status struct {
		Interfaces []struct {
			IPAddress string `json:"ipAddress"`
		} `json:"interfaces"`
	} `json:"status"`
}

func (p Provider) CreateVM(ctx context.Context, req domainvm.CreateVMRequest) (domainvm.VMDetail, error) {
	manifest, err := p.writeManifest(req)
	if err != nil {
		return domainvm.VMDetail{}, err
	}
	defer os.Remove(manifest)
	if _, err := p.runner.Run(ctx, "kubectl", "-n", p.namespace, "apply", "-f", manifest); err != nil {
		return domainvm.VMDetail{}, err
	}
	return p.GetVM(ctx, req.Name)
}

func (p Provider) ListVMs(ctx context.Context) ([]domainvm.VMSummary, error) {
	vmOut, err := p.runner.Run(ctx, "kubectl", "-n", p.namespace, "get", "virtualmachines", "-o", "json")
	if err != nil {
		return nil, err
	}
	vmiOut, err := p.runner.Run(ctx, "kubectl", "-n", p.namespace, "get", "virtualmachineinstances", "-o", "json")
	if err != nil {
		return nil, err
	}
	var vms vmListResponse
	var vmis vmiListResponse
	if err := json.Unmarshal([]byte(vmOut), &vms); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(vmiOut), &vmis); err != nil {
		return nil, err
	}
	ipByName := map[string]string{}
	for _, item := range vmis.Items {
		if len(item.Status.Interfaces) > 0 {
			ipByName[item.Metadata.Name] = item.Status.Interfaces[0].IPAddress
		}
	}
	items := make([]domainvm.VMSummary, 0, len(vms.Items))
	for _, item := range vms.Items {
		items = append(items, domainvm.VMSummary{
			Name:      item.Metadata.Name,
			Phase:     item.Status.PrintableStatus,
			Image:     item.Spec.Template.Metadata.Annotations["anarchy.io/image"],
			PrivateIP: ipByName[item.Metadata.Name],
		})
	}
	return items, nil
}

func (p Provider) GetVM(ctx context.Context, name string) (domainvm.VMDetail, error) {
	vmOut, err := p.runner.Run(ctx, "kubectl", "-n", p.namespace, "get", "virtualmachine", name, "-o", "json")
	if err != nil {
		return domainvm.VMDetail{}, err
	}
	vmiOut, err := p.runner.Run(ctx, "kubectl", "-n", p.namespace, "get", "virtualmachineinstance", name, "-o", "json")
	if err != nil {
		vmiOut = `{"status":{"interfaces":[]}}`
	}
	var vm vmSingleResponse
	var vmi vmiSingleResponse
	if err := json.Unmarshal([]byte(vmOut), &vm); err != nil {
		return domainvm.VMDetail{}, err
	}
	if err := json.Unmarshal([]byte(vmiOut), &vmi); err != nil {
		return domainvm.VMDetail{}, err
	}
	privateIP := ""
	if len(vmi.Status.Interfaces) > 0 {
		privateIP = vmi.Status.Interfaces[0].IPAddress
	}
	network := ""
	if len(vm.Spec.Template.Spec.Networks) > 0 {
		network = vm.Spec.Template.Spec.Networks[0].Name
	}
	return domainvm.VMDetail{
		Name:      vm.Metadata.Name,
		Phase:     vm.Status.PrintableStatus,
		Image:     vm.Spec.Template.Metadata.Annotations["anarchy.io/image"],
		CPU:       vm.Spec.Template.Spec.Domain.CPU.Cores,
		Memory:    vm.Spec.Template.Spec.Domain.Resources.Requests.Memory,
		Network:   network,
		PrivateIP: privateIP,
	}, nil
}

func (p Provider) StartVM(ctx context.Context, name string) error {
	_, err := p.runner.Run(ctx, "kubectl", "-n", p.namespace, "patch", "virtualmachine", name, "--type", "merge", "-p", `{"spec":{"running":true}}`)
	return err
}
func (p Provider) StopVM(ctx context.Context, name string) error {
	_, err := p.runner.Run(ctx, "kubectl", "-n", p.namespace, "patch", "virtualmachine", name, "--type", "merge", "-p", `{"spec":{"running":false}}`)
	return err
}
func (p Provider) RestartVM(ctx context.Context, name string) error {
	_, err := p.runner.Run(ctx, "kubectl", "-n", p.namespace, "restart", "vm", name)
	return err
}
func (p Provider) DeleteVM(ctx context.Context, name string) error {
	_, err := p.runner.Run(ctx, "kubectl", "-n", p.namespace, "delete", "virtualmachine", name)
	return err
}

func (p Provider) writeManifest(req domainvm.CreateVMRequest) (string, error) {
	file, err := os.CreateTemp("", "anarchy-vm-*.yaml")
	if err != nil {
		return "", err
	}
	rootDiskName := req.Name + "-rootdisk"
	networkName := req.Network
	if req.SubnetRef != "" {
		networkName = req.SubnetRef
	}
	manifest := fmt.Sprintf(`apiVersion: kubevirt.io/v1
kind: VirtualMachine
metadata:
  name: %s
  namespace: %s
spec:
  running: true
  dataVolumeTemplates:
    - metadata:
        name: %s
        annotations:
          cdi.kubevirt.io/storage.bind.immediate.requested: "true"
      spec:
        sourceRef:
          kind: DataSource
          name: %s
          namespace: %s
        storage:
          accessModes:
            - ReadWriteOnce
          resources:
            requests:
              storage: 20Gi
  template:
    metadata:
      annotations:
        anarchy.io/image: %s
    spec:
      domain:
        cpu:
          cores: %d
        resources:
          requests:
            memory: %s
        devices:
          disks:
            - name: rootdisk
              disk:
                bus: virtio
          interfaces:
            - name: %s
              masquerade: {}
      networks:
        - name: %s
          pod: {}
      volumes:
        - name: rootdisk
          dataVolume:
            name: %s
`, req.Name, p.namespace, rootDiskName, req.Image, p.namespace, req.Image, req.CPU, req.Memory, networkName, networkName, rootDiskName)
	if _, err := file.WriteString(manifest); err != nil {
		file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return file.Name(), nil
}
