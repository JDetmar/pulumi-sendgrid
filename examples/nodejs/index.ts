import * as pulumi from "@pulumi/pulumi";
import * as sendgrid from "@jdetmar/pulumi-sendgrid";

const myApiKey = new sendgrid.ApiKey("myApiKey", {
  name: "my-app-api-key",
  scopes: ["mail.send", "alerts.read"],
});

const myTemplate = new sendgrid.Template("myTemplate", {
  name: "welcome-email",
  generation: "dynamic",
});

const myEventWebhook = new sendgrid.EventWebhook("myEventWebhook", {
  url: "https://example.com/webhooks/sendgrid",
  friendlyName: "My App Webhook",
  enabled: false,
  delivered: true,
  open: true,
  click: true,
  bounce: true,
  dropped: true,
  spamReport: true,
});

const myDomainAuth = new sendgrid.DomainAuthentication("myDomainAuth", {
  domain: "example.com",
  automaticSecurity: true,
});

export const apiKeyId = myApiKey.apiKeyId;
export const templateId = myTemplate.templateId;
export const webhookId = myEventWebhook.webhookId;
export const domainId = myDomainAuth.domainId;
