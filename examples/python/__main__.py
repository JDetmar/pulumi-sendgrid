import pulumi
import pulumi_provider_sendgrid as sendgrid

my_random_resource = sendgrid.Random("myRandomResource", length=24)
my_random_component = sendgrid.RandomComponent("myRandomComponent", length=24)
pulumi.export("output", {
    "value": my_random_resource.result,
})
