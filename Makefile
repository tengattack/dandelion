NAME=dandelion
VERSION=0.0.3
REGISTRY_PREFIX=$(if $(REGISTRY),$(addsuffix /, $(REGISTRY)))

.PHONY: build publish

web-dep:
	cd web && npm i

web:
	cd web && npm run clean && npm run build
	cd .. && go generate ./...


build:
	docker build --build-arg version=${VERSION} \
		-t ${NAME}:${VERSION} .

publish:
	docker tag ${NAME}:${VERSION} ${REGISTRY_PREFIX}${NAME}:${VERSION}
	docker push ${REGISTRY_PREFIX}${NAME}:${VERSION}
