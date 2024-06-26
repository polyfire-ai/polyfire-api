GOARCH = amd64

UNAME = $(shell uname -s)

VERSION = $(shell cat VERSION)

BIN_NAME = server_start

BUILD_DIRECTORY = build

CODEGEN_DIRECTORY = codegen

ifndef OS
	ifeq ($(UNAME), Linux)
		OS = linux
	else ifeq ($(UNAME), Darwin)
		OS = darwin
	endif
endif

.DEFAULT_GOAL := all

all: $(CODEGEN_DIRECTORY)/openrouter-models.go fmt $(BUILD_DIRECTORY)/$(BIN_NAME)

$(BUILD_DIRECTORY)/$(BIN_NAME): codegen api.go ./**/*.go
	mkdir -p $(BUILD_DIRECTORY)
	GOOS=$(OS) GOARCH="$(GOARCH)" go build -o $(BUILD_DIRECTORY)/$(BIN_NAME) api.go

app.yaml: app.dev.yaml check-env
	@cat app.dev.yaml \
	| sed "s/{{APP_SERVICE_NAME}}/${APP_SERVICE_NAME}/" \
	| sed "s#{{SUPABASE_URL}}#${SUPABASE_URL}#" \
	| sed "s/{{SUPABASE_KEY}}/${SUPABASE_KEY}/" \
	| sed "s/{{OPENAI_API_KEY}}/${OPENAI_API_KEY}/" \
	| sed "s/{{OPENROUTER_API_KEY}}/${OPENROUTER_API_KEY}/" \
	| sed "s/{{COHERE_API_KEY}}/${COHERE_API_KEY}/" \
	| sed "s/{{OPENAI_ORGANIZATION}}/${OPENAI_ORGANIZATION}/" \
	| sed "s/{{POSTHOG_API_KEY}}/${POSTHOG_API_KEY}/" \
	| sed "s/{{REPLICATE_API_KEY}}/${REPLICATE_API_KEY}/" \
	| sed "s#{{LLAMA_URL}}#${LLAMA_URL}#" \
	| sed "s#{{POSTGRES_URI}}#${POSTGRES_URI}#" \
	| sed "s#{{API_URL}}#${API_URL}#" \
	| sed 's/{{ELEVENLABS_API_KEY}}/${ELEVENLABS_API_KEY}/' \
	| sed 's/{{DEEPGRAM_API_KEY}}/${DEEPGRAM_API_KEY}/' \
	| sed 's/{{GCS_PROJECT_ID}}/${GCS_PROJECT_ID}/' \
	| sed 's/{{GCS_BUCKET_NAME}}/${GCS_BUCKET_NAME}/' \
	| sed 's/{{ASSEMBLYAI_API_KEY}}/${ASSEMBLYAI_API_KEY}/' \
	| sed "s/{{JWT_SECRET}}/${JWT_SECRET}/" > app.yaml

check-env:
ifndef APP_SERVICE_NAME
	$(error APP_SERVICE_NAME is undefined)
endif
ifndef SUPABASE_URL
	$(error SUPABASE_URL is undefined)
endif
ifndef SUPABASE_KEY
	$(error SUPABASE_KEY is undefined)
endif
ifndef OPENAI_API_KEY
	$(error OPENAI_API_KEY is undefined)
endif
ifndef OPENROUTER_API_KEY
	$(error OPENROUTER_API_KEY is undefined)
endif
ifndef COHERE_API_KEY
	$(error COHERE_API_KEY is undefined)
endif
ifndef OPENAI_ORGANIZATION
	$(error OPENAI_ORGANIZATION is undefined)
endif
ifndef POSTHOG_API_KEY
	$(error POSTHOG_API_KEY is undefined)
endif
ifndef REPLICATE_API_KEY
	$(error REPLICATE_API_KEY is undefined)
endif
ifndef JWT_SECRET
	$(error JWT_SECRET is undefined)
endif
ifndef POSTGRES_URI
	$(error POSTGRES_URI is undefined)
endif
ifndef LLAMA_URL
	$(error LLAMA_URL is undefined)
endif
ifndef API_URL
	$(error API_URL is undefined)
endif
ifndef ELEVENLABS_API_KEY
	$(error ELEVENLABS_API_KEY is undefined)
endif
ifndef DEEPGRAM_API_KEY
	$(error DEEPGRAM_API_KEY is undefined)
endif
ifndef GCS_BUCKET_NAME
	$(error GCS_BUCKET_NAME is undefined)
endif
ifndef GCS_PROJECT_ID
	$(error GCS_PROJECT_ID is undefined)
endif
ifndef ASSEMBLYAI_API_KEY
	$(error ASSEMBLYAI_API_KEY is undefined)
endif

gcs-service-account.json:
ifndef GCS_SERVICE_ACCOUNT
	$(error GCS_SERVICE_ACCOUNT is undefined)
endif
	@echo ${GCS_SERVICE_ACCOUNT} | base64 -d > gcs-service-account.json

deploy: app.yaml codegen gcs-service-account.json
	gcloud app deploy --quiet --version v1-1

clean:
	rm -rf $(BUILD_DIRECTORY) app.yaml $(CODEGEN_DIRECTORY)

fmt:
	go fmt $$(go list ./...)

test-%: codegen
	@echo "TEST: $(shell echo $@ | sed s/^test-// | sed 's/--/\//')/"
	@cd $(shell echo $@ | sed s/^test-// | sed 's/--/\//') && go test -v && cd ..

TESTS = $(shell find | grep _test.go | xargs dirname | uniq | sed 's/\.\//test-/' | sed 's/\//--/g')

test: ${TESTS}

schema.sql:
	( pg_dump -Osx ${SUPABASE_PG_URI} && pg_dump -at models ${SUPABASE_PG_URI} ) > schema.sql

create-dev-db: schema.sql
	psql ${POSTGRES_URI} -f schema.sql
	printf "INSERT INTO public.auth_users (id) VALUES ('12345678-9101-1121-8141-516171819202');	INSERT INTO public.projects (id, name, auth_id, free_user_init, slug, allow_anonymous_auth, dev_rate_limit) VALUES ('98765432-1012-3456-889a-987654321012', 'Default Project', '12345678-9101-1121-8141-516171819202', true, 'default', true, false);	INSERT INTO public.projects (id, name, auth_id, free_user_init, slug, allow_anonymous_auth, dev_rate_limit) VALUES ('00000000-0000-0000-0000-000000000000', '', '12345678-9101-1121-8141-516171819202', false, '', false, false); INSERT INTO auth.users (id, email) VALUES ('12345678-9101-1121-8141-516171819202', 'example@example.com');" | psql ${POSTGRES_URI}

$(CODEGEN_DIRECTORY)/openrouter-models.json:
	mkdir -p $(CODEGEN_DIRECTORY)
	curl -s "https://openrouter.ai/api/v1/models" > $(CODEGEN_DIRECTORY)/openrouter-models.json

$(CODEGEN_DIRECTORY)/openrouter-models.csv: $(CODEGEN_DIRECTORY)/openrouter-models.json
	printf "model,provider,credit_input,credit_type,type,credit_output,image_url,official_name,hidden,option_stream,option_temperature,option_stop" > $(CODEGEN_DIRECTORY)/openrouter-models.csv
	cat codegen/openrouter-models.json  | jq -r '.data[] | select(.id != "openrouter/auto") | .id+",openrouter,"+(((.pricing.prompt|tonumber)/0.0000001|ceil)|tostring)+",token_input_output,completion,"+(((.pricing.completion|tonumber)/0.0000001|ceil)|tostring)+",/openrouter.webp,OpenRouter,false,true,true,true"' >> $(CODEGEN_DIRECTORY)/openrouter-models.csv

$(CODEGEN_DIRECTORY)/openrouter-models.go: $(CODEGEN_DIRECTORY)/openrouter-models.json
	printf "// This code has been generated automatically, any change made here will be lost \npackage codegen\n\nfunc OpenRouterPrices(model string, inputTokenCount int, outputTokenCount int) int {\n\tswitch model {" > $(CODEGEN_DIRECTORY)/openrouter-models.go
	cat $(CODEGEN_DIRECTORY)/openrouter-models.json | jq -r '.data[] | select(.id != "openrouter/auto") | "\tcase \""+ .id +"\":\n\t\treturn (inputTokenCount * "+(((.pricing.prompt|tonumber) / 0.0000001)|ceil|tostring)+") + (outputTokenCount * "+(((.pricing.completion|tonumber) / 0.0000001)|ceil|tostring)+")"' >> $(CODEGEN_DIRECTORY)/openrouter-models.go
	printf "\t}\n\treturn 0\n}\n\nfunc IsOpenRouterModel(model string) bool {\n\tswitch model {" >> $(CODEGEN_DIRECTORY)/openrouter-models.go
	cat $(CODEGEN_DIRECTORY)/openrouter-models.json | jq -r '.data[] | select(.id != "openrouter/auto") | "\tcase \""+ .id +"\":\n\t\treturn true"' >> $(CODEGEN_DIRECTORY)/openrouter-models.go
	 printf "\t}\n\t return false\n}" >> $(CODEGEN_DIRECTORY)/openrouter-models.go

update-openrouter-models: check-env $(CODEGEN_DIRECTORY)/openrouter-models.csv
	psql ${POSTGRES_URI} -f scripts/update_openrouter_models.sql

codegen: $(CODEGEN_DIRECTORY)/openrouter-models.go

.PHONY: clean fmt check-env deploy create-dev-db update-openrouter-models codegen test
