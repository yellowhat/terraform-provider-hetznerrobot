# [Hetzner Robot](https://registry.terraform.io/providers/yellowhat/hetznerrobot)

Manage Hetzner Robot via terraform.

Some references:

* https://github.com/dd-hetzner-robot/terraform-provider-dd-hetzner-robot
* https://github.com/silenium-dev/terraform-provider-hetzner-robot
* https://github.com/floshodan/hrobot-go/tree/main

## Regenerate docs

```bash
cd tools
go generate ./...
```

## Release and publish to the Terraform registry

1. Generate GPG Key

    ```bash
    apk add gpg gpg-agent

    gpg \
        --batch \
        --passphrase '' \
        --quick-gen-key \
        test@dummy.com \
        rsa4096 default 10y

    # GPG_PUBLIC_KEY
    gpg --export --armor test@dummy.com

    # GPG_PRIVATE_KEY
    gpg --export-secret-key --armor test@dummy.com
    ```

2. Sign in to [registry.terraform.io](https://registry.terraform.io) using GitHub

3. Go to `Settings` > `New GPG Key`:
    * `Namespace`: `yellowhat`
    * `ASCII Armor`: copy `GPG_PUBLIC_KEY`
    * `Source`: `GitHub`
    * `Source URL`: `https://github.com/yellowhat/terraform-provider-hetzner-robot`

4. Make at least 1 GitHub release

5. Go to `Publish` > `Provider` > Select organization > Select repository
