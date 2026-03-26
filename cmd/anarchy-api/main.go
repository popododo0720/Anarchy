package main

import (
	"fmt"
	"net/http"
	"os"

	diagkube "github.com/popododo0720/anarchy/internal/adapters/diagnose/kubernetes"
	imagekube "github.com/popododo0720/anarchy/internal/adapters/image/kubernetes"
	kexec "github.com/popododo0720/anarchy/internal/adapters/kubernetes/exec"
	nodekube "github.com/popododo0720/anarchy/internal/adapters/node/kubernetes"
	systemkube "github.com/popododo0720/anarchy/internal/adapters/system/kubernetes"
	vmkube "github.com/popododo0720/anarchy/internal/adapters/vm/kubernetes"
	appdiag "github.com/popododo0720/anarchy/internal/application/diagnose"
	appimage "github.com/popododo0720/anarchy/internal/application/image"
	appnode "github.com/popododo0720/anarchy/internal/application/node"
	appsystem "github.com/popododo0720/anarchy/internal/application/system"
	appvm "github.com/popododo0720/anarchy/internal/application/vm"
	httpdiag "github.com/popododo0720/anarchy/internal/transport/http/diagnose"
	httpimage "github.com/popododo0720/anarchy/internal/transport/http/image"
	httpnode "github.com/popododo0720/anarchy/internal/transport/http/node"
	httpsystem "github.com/popododo0720/anarchy/internal/transport/http/system"
	httpvm "github.com/popododo0720/anarchy/internal/transport/http/vm"
)

func main() {
	addr := os.Getenv("ANARCHY_API_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	namespace := os.Getenv("ANARCHY_NAMESPACE")
	if namespace == "" {
		namespace = "anarchy-system"
	}

	runner := kexec.NewCommandRunner()
	systemHandler := httpsystem.NewHandler(appsystem.NewService(systemkube.NewProvider(runner)))
	nodeHandler := httpnode.NewHandler(appnode.NewService(nodekube.NewProvider(runner)))
	diagnoseHandler := httpdiag.NewHandler(appdiag.NewService(diagkube.NewProvider(runner, namespace)))
	imageHandler := httpimage.NewHandler(appimage.NewService(imagekube.NewProvider(runner, namespace)))
	vmHandler := httpvm.NewHandler(appvm.NewService(vmkube.NewProvider(runner, namespace)))

	mux := http.NewServeMux()
	systemHandler.RegisterRoutes(mux)
	nodeHandler.RegisterRoutes(mux)
	diagnoseHandler.RegisterRoutes(mux)
	imageHandler.RegisterRoutes(mux)
	vmHandler.RegisterRoutes(mux)

	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
