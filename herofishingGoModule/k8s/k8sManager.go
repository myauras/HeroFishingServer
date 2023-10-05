package k8s

import (
	"context"
	logger "herofishingGoModule/logger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	log "github.com/sirupsen/logrus"
)

// 傳入服務namespace與name取得LoadBalancer分配的外部IP
func GetLoadBalancerExternalIP(servicesNameSpace string, servicesName string) (string, error) {
	// 建立K8s配置
	config, err := rest.InClusterConfig()
	if err != nil {
		return "", err
	}

	// 建立k8s client
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", err
	}

	// 取得Service
	// service, err := clientset.CoreV1().Services("herofishing-matchserver").Get(context.TODO(), "herofishing-matchmaker", metav1.GetOptions{})
	service, err := clientset.CoreV1().Services(servicesNameSpace).Get(context.TODO(), servicesName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	// 取得LoadBalancer的外部IP
	for _, ingress := range service.Status.LoadBalancer.Ingress {
		if ingress.IP != "" {
			log.Infof("%s ingress.IP: %s", logger.LOG_K8s, ingress.IP)
			return ingress.IP, nil
		}
	}

	return "", nil
}
