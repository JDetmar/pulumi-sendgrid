using System.Collections.Generic;
using System.Linq;
using Pulumi;
using Sendgrid = Pulumi.Sendgrid;

return await Deployment.RunAsync(() => 
{
    var myRandomResource = new Sendgrid.Random("myRandomResource", new()
    {
        Length = 24,
    });

    var myRandomComponent = new Sendgrid.RandomComponent("myRandomComponent", new()
    {
        Length = 24,
    });

    return new Dictionary<string, object?>{};
});

