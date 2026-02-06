import pulumi
import pulumi_sendgrid as sendgrid

my_api_key = sendgrid.ApiKey("myApiKey",
    name="my-app-api-key",
    scopes=["mail.send", "alerts.read"],
)

my_template = sendgrid.Template("myTemplate",
    name="welcome-email",
    generation="dynamic",
)

my_event_webhook = sendgrid.EventWebhook("myEventWebhook",
    url="https://example.com/webhooks/sendgrid",
    friendly_name="My App Webhook",
    enabled=False,
    delivered=True,
    open=True,
    click=True,
    bounce=True,
    dropped=True,
    spam_report=True,
)

my_domain_auth = sendgrid.DomainAuthentication("myDomainAuth",
    domain="example.com",
    automatic_security=True,
)

pulumi.export("apiKeyId", my_api_key.api_key_id)
pulumi.export("templateId", my_template.template_id)
pulumi.export("webhookId", my_event_webhook.webhook_id)
pulumi.export("domainId", my_domain_auth.domain_id)
