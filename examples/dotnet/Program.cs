using System.Collections.Generic;
using Pulumi;
using Sendgrid = Pulumi.Sendgrid;

return await Deployment.RunAsync(() =>
{
    var myApiKey = new Sendgrid.ApiKey("myApiKey", new()
    {
        Name = "my-app-api-key",
        Scopes = new[] { "mail.send", "alerts.read" },
    });

    var myTemplate = new Sendgrid.Template("myTemplate", new()
    {
        Name = "welcome-email",
        Generation = "dynamic",
    });

    var myEventWebhook = new Sendgrid.EventWebhook("myEventWebhook", new()
    {
        Url = "https://example.com/webhooks/sendgrid",
        FriendlyName = "My App Webhook",
        Enabled = false,
        Delivered = true,
        Open = true,
        Click = true,
        Bounce = true,
        Dropped = true,
        SpamReport = true,
    });

    var myDomainAuth = new Sendgrid.DomainAuthentication("myDomainAuth", new()
    {
        Domain = "example.com",
        AutomaticSecurity = true,
    });

    return new Dictionary<string, object?>
    {
        ["apiKeyId"] = myApiKey.ApiKeyId,
        ["templateId"] = myTemplate.TemplateId,
        ["webhookId"] = myEventWebhook.WebhookId,
        ["domainId"] = myDomainAuth.DomainId,
    };
});
