import * as pulumi from "@pulumi/pulumi";
import * as sendgrid from "@jdetmar/pulumi-sendgrid";

const myRandomResource = new sendgrid.Random("myRandomResource", {
  length: 24,
});
const myRandomComponent = new sendgrid.RandomComponent("myRandomComponent", {
  length: 24,
});
export const output = {
  value: myRandomResource.result,
};
