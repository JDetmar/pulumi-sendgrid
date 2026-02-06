# Pulumi SendGrid Provider

A native Pulumi provider for managing [SendGrid](https://sendgrid.com/) resources.

## Installation

### Node.js (npm)

```bash
npm install @jdetmar/pulumi-sendgrid
```

### Python (pip)

```bash
pip install pulumi-sendgrid
```

### Go

```bash
go get github.com/JDetmar/pulumi-sendgrid/sdk/go/sendgrid
```

### .NET (NuGet)

```bash
dotnet add package Community.Pulumi.Sendgrid
```

## Configuration

| Key | Environment Variable | Required | Description |
|-----|---------------------|----------|-------------|
| `sendgrid:apiKey` | `SENDGRID_API_KEY` | Yes | SendGrid API key for authentication |
| `sendgrid:baseUrl` | — | No | API base URL (default: `https://api.sendgrid.com`). Use `https://api.eu.sendgrid.com` for EU regional subusers. |

```bash
pulumi config set sendgrid:apiKey --secret SG.xxxxx
# or
export SENDGRID_API_KEY="SG.xxxxx"
```

## Example (TypeScript)

```typescript
import * as sendgrid from "@jdetmar/pulumi-sendgrid";

const apiKey = new sendgrid.ApiKey("myApiKey", {
  name: "my-app-api-key",
  scopes: ["mail.send", "alerts.read"],
});

const template = new sendgrid.Template("myTemplate", {
  name: "welcome-email",
  generation: "dynamic",
});

export const apiKeyId = apiKey.apiKeyId;
export const templateId = template.templateId;
```

See the [`examples/`](./examples/) directory for complete programs in Go, Python, .NET, and YAML.

## Resources

| Resource | Description |
|----------|-------------|
| `sendgrid:Alert` | Email alerts for usage and statistics thresholds |
| `sendgrid:ApiKey` | API keys with scoped permissions |
| `sendgrid:DomainAuthentication` | Domain authentication (DKIM/SPF) for sender identity |
| `sendgrid:EventWebhook` | Webhooks for email event notifications |
| `sendgrid:GlobalSuppression` | Global unsubscribe entries |
| `sendgrid:IpPool` | IP pools for organizing dedicated IPs (Pro plan) |
| `sendgrid:LinkBranding` | Branded tracking links for click/open tracking |
| `sendgrid:Subuser` | Subuser accounts with independent settings (Pro plan) |
| `sendgrid:Teammate` | Teammate accounts with role-based access |
| `sendgrid:Template` | Transactional email templates |
| `sendgrid:TemplateVersion` | Versioned content for email templates |
| `sendgrid:UnsubscribeGroup` | Suppression groups for subscription management |
| `sendgrid:VerifiedSender` | Verified sender identities |

## Development

### Prerequisites

- [Go 1.24+](https://golang.org/dl/)
- [Node.js](https://nodejs.org/)
- [Python 3](https://www.python.org/)
- [.NET SDK](https://dotnet.microsoft.com/download)
- [`pulumictl`](https://github.com/pulumi/pulumictl#installation)
- [Pulumi CLI](https://www.pulumi.com/docs/install/)

### Build and install

```bash
make build install
```

### Run provider unit tests

```bash
make test_provider
```

### Regenerate SDKs after provider changes

```bash
make codegen
```

## License

Apache 2.0 — see [LICENSE](./LICENSE) for details.
