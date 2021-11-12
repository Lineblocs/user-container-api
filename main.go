package main
import (
	"fmt"
	"log"
	"os"
	"net/http"
	"context"
	"github.com/gorilla/mux"
	"github.com/gorilla/handlers"
	//batchv1 "k8s.io/client-go/applyconfigurations/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//batchv1 "k8s.io/client-go/applyconfigurations/batch/v1"
	v1 "k8s.io/api/core/v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/kubernetes"
	rest "k8s.io/client-go/rest"
	"k8s.io/apimachinery/pkg/util/intstr"
	_ "k8s.io/client-go/tools/clientcmd"
	_ "k8s.io/client-go/util/homedir"
		"k8s.io/client-go/util/retry"
)

func createK8SConfig() (*rest.Config, error) {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return config, nil
}

func launchK8sResources(clientset *kubernetes.Clientset, userId string, name string, workspaceId string) (error) {
	fmt.Println("launching K8s resources for " + name)
	fmt.Println("USER ID = " + userId)
	token :=""
	secret :=""
	image := "754569496111.dkr.ecr.ca-central-1.amazonaws.com/lineblocs-k8s-user:latest"
	namespace := "voip-users"
	svcName := name
	domain := name+".lineblocs.com"
	servicesClient := clientset.CoreV1().Services(namespace)
	svcPort := intstr.FromInt(10000)
	service := &v1.Service{
            ObjectMeta: metav1.ObjectMeta{
                Name:                       svcName,
                Namespace:                  namespace,
                Labels: map[string]string{
                    "app": name,
                },
            },
            Spec: v1.ServiceSpec{
				Ports: []v1.ServicePort{
					{
						Port: 10000,
						TargetPort: svcPort,
					},
				},
                Selector:                 map[string]string{
					"app": name,
				},
                ClusterIP:                "",

            },
   	}

	deploymentsClient := clientset.AppsV1().Deployments(namespace)

	var replicas int32 = 1
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": name,
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": name,
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  name,
							Image: image,
							Ports: []v1.ContainerPort{
								{
									Name:          "http",
									Protocol:      v1.ProtocolTCP,
									ContainerPort: 10000,
								},
							},
							Env: []v1.EnvVar{
								{
									Name: "LINEBLOCS_TOKEN",
									Value: token,
								},
								{
									Name: "LINEBLOCS_SECRET",
									Value: secret,
								},
								{
									Name: "LINEBLOCS_WORKSPACE_ID",
									Value: workspaceId,
								},
								{
									Name: "LINEBLOCS_USER_ID",
									Value: userId,
								},
								{
									Name: "LINEBLOCS_DOMAIN",
									Value: domain,
								},
							},
						},
					},
                    RestartPolicy: v1.RestartPolicyNever,
					ImagePullSecrets: []v1.LocalObjectReference{
						{
							Name: "lineblocs-regcred",
						},
					},
				},
			},
		},
	}

	// Create Service
	fmt.Println("Creating service...")
	opts := metav1.CreateOptions{}
	ctx := context.Background()
	_, err := servicesClient.Create(ctx, service, opts)
	if err != nil {
		return err
	}
	// Create Deployment
	fmt.Println("Creating deployment...")
	opts2 := metav1.CreateOptions{}
	ctx2 := context.Background()
	_, err= deploymentsClient.Create(ctx2, deployment, opts2)
	if err != nil {
		return err
	}
	return nil
}

func CreateContainer(w http.ResponseWriter, r *http.Request) {
	fmt.Println("create container called..")
	workspace := r.FormValue("workspace")
	workspaceId := r.FormValue("workspace_id")
	userId:= r.FormValue("user_id")
	cfg, err:= createK8SConfig()
	if err != nil {
		fmt.Printf("error occured")
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		fmt.Printf("error occured")
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}


	err = launchK8sResources(clientset, userId, workspace, workspaceId)
	if err != nil {
		fmt.Printf("error occured")
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func updateDeployment(clientset *kubernetes.Clientset, name string) (error) {
	namespace := "voip-users"
	deploymentsClient := clientset.AppsV1().Deployments(namespace)
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Retrieve the latest version of Deployment before attempting update
		// RetryOnConflict uses exponential backoff to avoid exhausting the apiserver
		result, getErr := deploymentsClient.Get(context.TODO(), name, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}

		result.Spec.Template.Spec.Containers[0].Image = "nginx:1.13" // change nginx version
		_, updateErr := deploymentsClient.Update(context.TODO(), result, metav1.UpdateOptions{})
		return updateErr
	})
	if err != nil {
		return err
	}
	return nil
}
func UpdateContainer(w http.ResponseWriter, r *http.Request) {
	fmt.Println("update container called..")
	workspace := r.FormValue("workspace")
	cfg, err:= createK8SConfig()
	if err != nil {
		fmt.Printf("error occured")
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		fmt.Printf("error occured")
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = updateDeployment(clientset, workspace)
	if err != nil {
		fmt.Printf("error occured")
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}



func main() {
    r := mux.NewRouter()
    // Routes consist of a path and a handler function.
	r.HandleFunc("/createContainer", CreateContainer).Methods("POST");
	r.HandleFunc("/updateContainer", UpdateContainer).Methods("POST");

	loggedRouter := handlers.LoggingHandler(os.Stdout, r)

	// Bind to a port and pass our router in
	fmt.Println("Starting server..")
    log.Fatal(http.ListenAndServe(":80", loggedRouter))
}	