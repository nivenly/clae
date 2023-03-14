Nivenly Contributor License Agreement Web Application
====

The **clae** project provides a web form to sign the Nivenly **Apache 2.0** and **Creative Commons BY-SA** Contributor License Agreement (CLA). It also serves a REST API, that is used by cla-bot (see https://finos.github.io/cla-bot/).

 - [Apache 2.0 License](https://www.apache.org/licenses/LICENSE-2.0) is used as our default license for software and source code.
 - [Creative Commons BY-SA](https://creativecommons.org/licenses/by-sa/4.0/) is used as our default license for documentation and user contributed media such as blogs and tutorials.

### How to redeploy?

If you have done changes to the code and pushed to main, GitHub Actions will build a new Docker container and upload it (do this via a pull request) to ghcr. Clae is deployed using a Kubernetes *Deployment* and a Kubernetes *Secret*. So after the GH Action has finished, you can redeploy by deleting the pod. It will reconcile and download the newest clae container.

### Adding @cla-bot to an Org

Setting up the cla robot should take about 5 minutes and requires a few button clicks in the GitHub dashboard. No code changes are required.

1. Install the [cla-bot app](https://github.com/apps/cla-bot) for the **organization**. 
2. Create a new repository called `clabot-config` in the **organization**. 
3. Create a new file called `.clabot` in the newly created repository with the following json.

```json
{
   "message":"In order to contribute to a Nivenly Foundation project you must [sign and agree to the CLA](https://cla.nivenly.org). Reply with `@cla-bot check` to check again.",
   "label":"Access Granted",
   "contributors": "https://cla.nivenly.org/contributor?token=barnaby-snacks-secret&checkContributor="
}
```

For the **about** dialogue we suggest the following.

```
Required for legal purposes. Do not remove.
```

### Environment Variables

`TOKEN` - specifies the authentication token you need to access `/dump?token=` and `/contributor?token=` 

`DATABASE` - specifies the filename of the sqlite database

### Token 

The token will need to be `base64` encoded to the secret. The current value of the token can be found in 1Password in the [nivenly-ops vault](https://hachyderm.1password.com/vaults/7eqql3ejerit3pt2g2r2hvj24q/allitems/lrtk32mvhsxgmb2alnh354m4zi0)

```bash 
echo -n "my-dirty-secret" | base64 
bXktZGlydHktc2VjcmV0
```

And set in the secret

```yaml 
apiVersion: v1
kind: Secret
metadata:
  name: clae-token
type: Opaque
data:
  TOKEN: bXktZGlydHktc2VjcmV0
```

Which can then be verified your token is deployed by accessing the API

```bash
curl -Sl https://hostname/dump?token=my-dirty-secret | jq
```

### API Endpoints

`GET /` - serves the CLA form

`POST /` - receives the submitted CLA form

`GET /contributor?token=my-dirty-secret&checkContributor=johndoe` - returns an info if the user has signed the CLA

 `GET /dump?token=my-dirty-secret` - returns a database dump in JSON format
