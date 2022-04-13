# User Service

To configure a bootstrap admin user account, create a secure AWS SSM Parameter named `/versionary/api-admin` that 
contains a known (secret) bearer token. You can use any bearer token you like, but they're generally TUIDs. An example 
AWS CLI command is provided below. Replace the `your-bearer-token` with a real token.

```bash
aws ssm put-parameter --name /versionary/api-admin --value your-bearer-token --type SecureString --overwrite
```
