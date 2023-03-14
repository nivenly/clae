NAMESPACE:=clae-nivenly

.PHONY: render
render:
	kubectl kustomize ./

.PHONY: apply
apply:
	kubectl apply -k ./ -n $(NAMESPACE)
