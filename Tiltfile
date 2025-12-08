# Tiltfile
load('ext://restart_process', 'docker_build_with_restart')
load('ext://secret', 'secret_yaml_tls')

# 1. Generate Certs locally
local('./scripts/gen-certs.sh')

# 2. Create Secret from generated files using secret extension
k8s_yaml(secret_yaml_tls(
    'archy-webhook-certs',
    'certs/tls.crt',
    'certs/tls.key'
))

# 3. Build and Deploy Webhook
docker_build_with_restart(
    'archy-webhook',
    '.',
    dockerfile='Containerfile.dev',
    # Live update for fast iteration
    live_update=[
        sync('.', '/app'),
        run('go build -o webhook ./cmd/webhook', trigger='./**/*.go'),
    ],
    entrypoint=['/app/webhook']
)

k8s_yaml('deploy/deployment.yaml')

# 4. Apply Webhook Configuration with CA Bundle
# Read CA cert and base64 encode it
ca_bundle = local('cat certs/ca.crt | base64 | tr -d "\n"')
# Use sed to patch the file and output to stdout (avoids Starlark string/blob issues)
config_yaml = local('sed "s|Cg==|%s|g" deploy/webhook-config.yaml' % ca_bundle)
k8s_yaml(config_yaml)

k8s_resource('archy-webhook', 
    port_forwards=8443,
)
