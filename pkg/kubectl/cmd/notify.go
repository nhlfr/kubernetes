package cmd

import (
	"fmt"
	"io"

	"github.com/golang/glog"
	"github.com/spf13/cobra"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/restclient"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/kubectl/resource"
	utilerrors "k8s.io/kubernetes/pkg/util/errors"
)

const (
	notify_example = ""
)

func NewCmdNotify(f *cmdutil.Factory, cmdIn io.Reader, cmdOut, cmdErr io.Writer) *cobra.Command {
	options := &NotifyOptions{
		In:  cmdIn,
		Out: cmdOut,
		Err: cmdErr,
	}
	cmd := &cobra.Command{
		Use:     "notify POD [-c container] NOTIFICATION",
		Short:   "Send a notification to a container.",
		Long:    "Send a notification to a container.",
		Example: notify_example,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(options.Complete(f, cmd, args))
			cmdutil.CheckErr(options.Validate())
			cmdutil.CheckErr(options.Run(f, cmd))
		},
	}
	cmd.Flags().StringVarP(&options.PodName, "pod", "p", "", "Pod name")
	// TODO support UID
	cmd.Flags().StringVarP(&options.ContainerName, "container", "c", "", "Container name. If omitted, the first container in the pod will be chosen")
	return cmd
}

type NotifyOptions struct {
	Namespace     string
	PodName       string
	ContainerName string
	Notification  string

	Pod *api.Pod

	In  io.Reader
	Out io.Writer
	Err io.Writer

	Builder *resource.Builder
	Client  *client.Client
	Config  *restclient.Config
}

func (p *NotifyOptions) Complete(f *cmdutil.Factory, cmd *cobra.Command, argsIn []string) error {
	switch len(argsIn) {
	case 0:
		return cmdutil.UsageError(cmd, "POD is required for notify")
	case 1:
		return cmdutil.UsageError(cmd, "NOTIFICATION is required for notify")
	}
	p.PodName = argsIn[0]
	p.Notification = argsIn[1]

	namespace, _, err := f.DefaultNamespace()
	if err != nil {
		return err
	}
	p.Namespace = namespace

	config, err := f.ClientConfig()
	if err != nil {
		return err
	}
	p.Config = config

	client, err := f.Client()
	if err != nil {
		return err
	}
	p.Client = client

	mapper, typer := f.Object(cmdutil.GetIncludeThirdPartyAPIs(cmd))
	p.Builder = resource.NewBuilder(mapper, typer, resource.ClientMapperFunc(f.ClientForMapping), f.Decoder(true)).
		ContinueOnError().
		NamespaceParam(namespace).DefaultNamespace().
		Flatten().
		Latest()

	return nil
}

func (p *NotifyOptions) Validate() error {
	allErrs := []error{}
	if len(p.PodName) == 0 {
		allErrs = append(allErrs, fmt.Errorf("pod name must be specified"))
	}
	if len(p.Notification) == 0 {
		allErrs = append(allErrs, fmt.Errorf("notification name must be specified"))
	}
	if p.Config == nil {
		allErrs = append(allErrs, fmt.Errorf("config must be provided"))
	}
	return utilerrors.NewAggregate(allErrs)
}

func (p *NotifyOptions) Run(f *cmdutil.Factory, cmd *cobra.Command) error {
	pod, err := p.Client.Pods(p.Namespace).Get(p.PodName)
	if err != nil {
		return err
	}

	containerName := p.ContainerName
	if len(containerName) == 0 {
		containerName = pod.Spec.Containers[0].Name
		glog.V(4).Infof("defaulting container name to %s", containerName)
	}

	// TODO: consider abstracting into a client invocation or client helper
	req := p.Client.RESTClient.Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("notify").
		Param("container", containerName).
		Param("notificationName", p.Notification)
	/* req.VersionedParams(&api.PodNotifyOptions{
		Container:        containerName,
		NotificationName: p.Notification,
	}, api.ParameterCodec) */

	return req.Do().Error()
}
