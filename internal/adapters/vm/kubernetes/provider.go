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

type vmNetwork struct {
	Name string `json:"name"`
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
					Networks []vmNetwork `json:"networks"`
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
				Networks []vmNetwork `json:"networks"`
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

type vmiInterface struct {
	Name      string `json:"name"`
	IPAddress string `json:"ipAddress"`
}

type vmiListResponse struct {
	Items []struct {
		Metadata struct {
			Name string `json:"name"`
		} `json:"metadata"`
		Status struct {
			Interfaces []vmiInterface `json:"interfaces"`
		} `json:"status"`
	} `json:"items"`
}

type vmiSingleResponse struct {
	Status struct {
		Interfaces []vmiInterface `json:"interfaces"`
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
	ifacesByName := map[string][]vmiInterface{}
	for _, item := range vmis.Items {
		ifacesByName[item.Metadata.Name] = item.Status.Interfaces
	}
	items := make([]domainvm.VMSummary, 0, len(vms.Items))
	for _, item := range vms.Items {
		network := ""
		if len(item.Spec.Template.Spec.Networks) > 0 {
			network = item.Spec.Template.Spec.Networks[0].Name
		}
		subnet := item.Spec.Template.Metadata.Annotations["anarchy.io/subnet"]
		if subnet == "" {
			subnet = network
		}
		attachments := attachmentDetails(item.Spec.Template.Spec.Networks, ifacesByName[item.Metadata.Name], item.Spec.Template.Metadata.Annotations)
		primaryIP := ""
		if len(attachments) > 0 {
			primaryIP = attachments[0].IPAddress
		}
		items = append(items, domainvm.VMSummary{
			Name:               item.Metadata.Name,
			Phase:              item.Status.PrintableStatus,
			Image:              item.Spec.Template.Metadata.Annotations["anarchy.io/image"],
			Network:            network,
			SubnetRef:          subnet,
			PrivateIP:          primaryIP,
			NetworkAttachments: attachments,
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
	attachments := attachmentDetails(vm.Spec.Template.Spec.Networks, vmi.Status.Interfaces, vm.Spec.Template.Metadata.Annotations)
	privateIP := ""
	if len(attachments) > 0 {
		privateIP = attachments[0].IPAddress
	}
	network := ""
	if len(vm.Spec.Template.Spec.Networks) > 0 {
		network = vm.Spec.Template.Spec.Networks[0].Name
	}
	subnet := vm.Spec.Template.Metadata.Annotations["anarchy.io/subnet"]
	if subnet == "" {
		subnet = network
	}
	return domainvm.VMDetail{
		Name:               vm.Metadata.Name,
		Phase:              vm.Status.PrintableStatus,
		Image:              vm.Spec.Template.Metadata.Annotations["anarchy.io/image"],
		CPU:                vm.Spec.Template.Spec.Domain.CPU.Cores,
		Memory:             vm.Spec.Template.Spec.Domain.Resources.Requests.Memory,
		Network:            network,
		SubnetRef:          subnet,
		PrivateIP:          privateIP,
		NetworkAttachments: attachments,
	}, nil
}

func attachmentDetails(networks []vmNetwork, ifaces []vmiInterface, annotations map[string]string) []domainvm.NetworkAttachment {
	if len(networks) == 0 {
		return nil
	}
	ipByName := map[string]string{}
	for _, iface := range ifaces {
		ipByName[iface.Name] = iface.IPAddress
	}
	items := make([]domainvm.NetworkAttachment, 0, len(networks))
	for i, net := range networks {
		subnet := net.Name
		if i == 0 {
			if ann := annotations["anarchy.io/subnet"]; ann != "" {
				subnet = ann
			}
		}
		name := net.Name
		role := annotations["anarchy.io/attachment."+name+".role"]
		if role == "" {
			if i == 0 {
				role = "external"
			} else {
				role = "internal"
			}
		}
		nadRef := annotations["anarchy.io/attachment."+name+".nad"]
		network := annotations["anarchy.io/attachment."+name+".network"]
		if network == "" {
			network = net.Name
		}
		itemSubnet := annotations["anarchy.io/attachment."+name+".subnet"]
		if itemSubnet == "" {
			itemSubnet = subnet
		}
		items = append(items, domainvm.NetworkAttachment{
			Name:      name,
			Network:   network,
			SubnetRef: itemSubnet,
			NADRef:    nadRef,
			Role:      role,
			IPAddress: ipByName[net.Name],
			Primary:   i == 0,
		})
	}
	return items
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
	attachments := req.NetworkAttachments
	if len(attachments) == 0 {
		fallback := domainvm.NetworkAttachment{Name: "nic0", Network: req.Network, SubnetRef: req.SubnetRef, Primary: true}
		attachments = []domainvm.NetworkAttachment{fallback}
	}
	primary := attachments[0]
	if !primary.Primary {
		for _, attachment := range attachments {
			if attachment.Primary {
				primary = attachment
				break
			}
		}
	}
	primarySubnet := primary.SubnetRef
	if primarySubnet == "" {
		primarySubnet = primary.Network
	}
	interfacesYAML := ""
	networksYAML := ""
	attachmentAnnotations := ""
	for i, attachment := range attachments {
		name := attachment.Name
		if name == "" {
			name = fmt.Sprintf("nic%d", i)
		}
		networkName := attachment.SubnetRef
		if networkName == "" {
			networkName = attachment.Network
		}
		role := attachment.Role
		if role == "" {
			if attachment.Primary {
				role = "external"
			} else {
				role = "internal"
			}
		}
		nadRef := attachment.NADRef
		if attachment.Primary {
			interfacesYAML += fmt.Sprintf("            - name: %s\n              masquerade: {}\n", name)
			networksYAML += fmt.Sprintf("        - name: %s\n          pod: {}\n", name)
		} else {
			if nadRef == "" {
				nadRef = networkName
			}
			interfacesYAML += fmt.Sprintf("            - name: %s\n              bridge: {}\n", name)
			networksYAML += fmt.Sprintf("        - name: %s\n          multus:\n            networkName: %s\n", name, nadRef)
		}
		attachmentAnnotations += fmt.Sprintf("        anarchy.io/attachment.%s.network: %s\n", name, attachment.Network)
		attachmentAnnotations += fmt.Sprintf("        anarchy.io/attachment.%s.subnet: %s\n", name, networkName)
		attachmentAnnotations += fmt.Sprintf("        anarchy.io/attachment.%s.role: %s\n", name, role)
		if nadRef != "" {
			attachmentAnnotations += fmt.Sprintf("        anarchy.io/attachment.%s.nad: %s\n", name, nadRef)
		}
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
        anarchy.io/subnet: %s
%s    spec:
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
%s      networks:
%s      volumes:
        - name: rootdisk
          dataVolume:
            name: %s
`, req.Name, p.namespace, rootDiskName, req.Image, p.namespace, req.Image, primarySubnet, attachmentAnnotations, req.CPU, req.Memory, interfacesYAML, networksYAML, rootDiskName)
	if _, err := file.WriteString(manifest); err != nil {
		file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return file.Name(), nil
}
