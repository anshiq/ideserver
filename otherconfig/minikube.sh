minikube delete && minikube start 
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/cloud/deploy.yaml
minikube addons enable ingress
minikube ip or minikube tunnel , to access ingress
ssh -i "awssubject.pem"  -R 8080:localhost:80 ubuntu@ec2-18-235-57-141.compute-1.amazonaws.com // 80 is port of ingress