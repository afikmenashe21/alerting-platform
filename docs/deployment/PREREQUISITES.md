# Prerequisites and Credentials Guide

Everything you need to deploy the alerting platform to AWS.

## Required: AWS Credentials

### Step 1: Create AWS Account (if you don't have one)

1. Go to https://aws.amazon.com
2. Click "Create an AWS Account"
3. Follow signup process
4. **Enable Free Tier** (automatic for first 12 months)

### Step 2: Create IAM User with Admin Access

⚠️ **Don't use root account credentials!** Create an IAM user instead.

#### Via AWS Console (Easiest):

1. Log in to AWS Console: https://console.aws.amazon.com
2. Go to **IAM** service
3. Click **Users** → **Create user**
4. User name: `terraform-deploy` (or your choice)
5. Enable **"Provide user access to AWS Management Console"** (optional)
6. Click **Next**
7. **Attach policies directly** → Select **"AdministratorAccess"**
   - (For production, use more restrictive policies)
8. Click **Next** → **Create user**
9. Click **"Create access key"**
10. Select **"Command Line Interface (CLI)"**
11. Click **Next** → **Create access key**
12. **IMPORTANT**: Copy and save:
    - **Access key ID**: `AKIAIOSFODNN7EXAMPLE`
    - **Secret access key**: `wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY`
    
    ⚠️ **You won't see the secret key again!**

#### Minimum Required IAM Permissions

If you want more restrictive access, the user needs:
- EC2 (full)
- ECS (full)
- RDS (full)
- ElastiCache (full)
- ECR (full)
- VPC (full)
- CloudWatch Logs (full)
- IAM (create roles/policies)
- Application Load Balancer (full)

### Step 3: Configure AWS CLI

```bash
# Install AWS CLI (if not installed)
# macOS:
brew install awscli

# Linux:
curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
unzip awscliv2.zip
sudo ./aws/install

# Windows:
# Download from: https://awscli.amazonaws.com/AWSCLIV2.msi

# Configure AWS CLI
aws configure

# Enter when prompted:
AWS Access Key ID: AKIAIOSFODNN7EXAMPLE
AWS Secret Access Key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
Default region name: us-east-1
Default output format: json
```

### Step 4: Verify AWS Access

```bash
# Test AWS credentials
aws sts get-caller-identity

# Expected output:
{
    "UserId": "AIDAI...",
    "Account": "123456789012",
    "Arn": "arn:aws:iam::123456789012:user/terraform-deploy"
}

# If this works, you're good to go! ✅
```

## Required: Database Password

You need to choose a strong password for RDS Postgres.

### Generate a Strong Password

```bash
# Option 1: Random password
openssl rand -base64 32

# Option 2: Use a password manager
# 1Password, LastPass, Bitwarden, etc.

# Example strong password:
# X9k$mP2vL#nQ8wR5tY7uE4aZ3cD6fG1h
```

⚠️ **Requirements**:
- At least 8 characters
- Include uppercase, lowercase, numbers, symbols
- Don't use common words
- Save it securely (password manager)

## Required: Install Tools

### 1. Terraform

```bash
# macOS
brew tap hashicorp/tap
brew install hashicorp/tap/terraform

# Linux (Ubuntu/Debian)
wget -O- https://apt.releases.hashicorp.com/gpg | sudo gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list
sudo apt update && sudo apt install terraform

# Windows (Chocolatey)
choco install terraform

# Verify
terraform version
# Should show: Terraform v1.5.0 or higher
```

### 2. Docker

```bash
# macOS
brew install --cask docker
# Then start Docker Desktop

# Linux (Ubuntu/Debian)
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER
# Log out and back in

# Windows
# Download from: https://www.docker.com/products/docker-desktop

# Verify
docker --version
docker ps  # Should not error
```

### 3. AWS CLI (Already installed above)

```bash
# Verify
aws --version
# Should show: aws-cli/2.x.x or higher
```

### 4. Git (Usually pre-installed)

```bash
# Verify
git --version

# If not installed:
# macOS: xcode-select --install
# Linux: sudo apt install git
# Windows: https://git-scm.com/download/win
```

## Optional: GitHub Setup (For CI/CD)

If you want automated deployments via GitHub Actions:

### Configure GitHub Secrets

1. Go to your GitHub repository
2. Click **Settings** → **Secrets and variables** → **Actions**
3. Click **New repository secret**
4. Add these secrets:

| Secret Name | Value | Example |
|-------------|-------|---------|
| `AWS_ACCESS_KEY_ID` | Your AWS access key ID | `AKIAIOSFODNN7EXAMPLE` |
| `AWS_SECRET_ACCESS_KEY` | Your AWS secret key | `wJalrXUtnFEMI/K...` |
| `AWS_REGION` | AWS region | `us-east-1` |
| `DB_PASSWORD` | Your RDS password | `X9k$mP2vL#nQ8...` |

## Configuration Checklist

Before deploying, make sure you have:

- [x] AWS account created
- [x] IAM user with admin access created
- [x] Access key ID and secret key saved securely
- [x] AWS CLI installed and configured (`aws configure`)
- [x] AWS credentials verified (`aws sts get-caller-identity`)
- [x] Terraform installed (`terraform version`)
- [x] Docker installed and running (`docker ps`)
- [x] Git installed (`git --version`)
- [x] Strong database password generated
- [x] Repository cloned locally

## Your Configuration File

Create `terraform/terraform.tfvars`:

```hcl
# For ULTRA-LOW-COST (recommended):
# cp terraform/terraform.tfvars.ultra-low-cost terraform/terraform.tfvars

# OR for standard deployment:
# cp terraform/terraform.tfvars.example terraform/terraform.tfvars

# Then edit:
vim terraform/terraform.tfvars

# Required changes:
db_password = "YOUR_STRONG_PASSWORD_HERE"  # ← Change this!

# Optional changes:
aws_region = "us-east-1"  # Change if you prefer another region
project_name = "alerting-platform"  # Change if you want different name
```

## Cost Estimates

### Ultra-Low-Cost Config (Recommended)
- **Free tier**: ~$5-10/month
- **After free tier**: ~$57/month

### Standard Config
- **Free tier**: ~$35-50/month
- **After free tier**: ~$100-110/month

## What You DON'T Need

❌ You don't need:
- Credit card with high limit (start with free tier)
- Domain name (uses ALB DNS)
- SSL certificate (can add later)
- Email service credentials (optional, for sender service)
- Slack webhook (optional, for notifications)
- Custom VPC (Terraform creates it)
- Existing database (Terraform creates RDS)
- Kafka cluster (Terraform creates it on ECS)

## Security Best Practices

### Protect Your Credentials

1. **Never commit credentials to Git**
   - `terraform.tfvars` is in `.gitignore` ✅
   - Don't share your access keys

2. **Use AWS Secrets Manager (Production)**
   ```bash
   # Store DB password in Secrets Manager
   aws secretsmanager create-secret \
     --name alerting-platform/db-password \
     --secret-string "YOUR_PASSWORD"
   ```

3. **Enable MFA on AWS Account**
   - Go to IAM → Your user → Security credentials
   - Enable MFA (Multi-Factor Authentication)

4. **Rotate Access Keys Regularly**
   - Every 90 days minimum
   - Create new key, update config, delete old key

5. **Use IAM Roles in Production**
   - Instead of access keys
   - More secure, no credentials to manage

## Troubleshooting Credentials

### "Unable to locate credentials"

```bash
# Check AWS config
cat ~/.aws/credentials
cat ~/.aws/config

# Re-configure
aws configure
```

### "Access Denied" Errors

```bash
# Check your user has correct permissions
aws iam get-user
aws iam list-attached-user-policies --user-name YOUR_USERNAME

# You should see AdministratorAccess policy
```

### "Region not set"

```bash
# Set default region
aws configure set region us-east-1

# Or export as env var
export AWS_REGION=us-east-1
export AWS_DEFAULT_REGION=us-east-1
```

## Ready to Deploy?

Once you have:
1. ✅ AWS credentials configured
2. ✅ Tools installed
3. ✅ Database password chosen
4. ✅ Configuration file created

Follow the deployment guide:
```bash
cat docs/deployment/QUICKSTART.md
```

Or for ultra-low-cost:
```bash
cat docs/deployment/ULTRA_LOW_COST.md
```

## Support

**AWS Free Tier Details**: https://aws.amazon.com/free/  
**AWS CLI Installation**: https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html  
**Terraform Installation**: https://developer.hashicorp.com/terraform/downloads  
**Docker Installation**: https://docs.docker.com/get-docker/

## Quick Start Command Summary

```bash
# 1. Install tools
brew install awscli hashicorp/tap/terraform
brew install --cask docker

# 2. Configure AWS
aws configure

# 3. Verify
aws sts get-caller-identity
terraform version
docker ps

# 4. Clone repo (if not done)
git clone <your-repo>
cd alerting-platform

# 5. Configure for ultra-low-cost
cd terraform
cp terraform.tfvars.ultra-low-cost terraform.tfvars
vim terraform.tfvars  # Set db_password

# 6. Deploy!
terraform init
terraform apply
```

You're ready! 🚀
