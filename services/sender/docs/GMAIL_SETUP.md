# Gmail SMTP Configuration

This guide shows how to configure the sender service to use Gmail SMTP.

## Quick Setup

Set these environment variables before running the sender service:

```bash
export SMTP_HOST=smtp.gmail.com
export SMTP_PORT=587
export SMTP_USER=your-email@gmail.com
export SMTP_PASSWORD=your-app-password
export SMTP_FROM=your-email@gmail.com
```

**Important**: Replace `your-email@gmail.com` with your actual Gmail address and `your-app-password` with a Gmail App Password (see below).

## Running with Gmail Configuration

```bash
# Set environment variables (use your actual credentials)
export SMTP_HOST=smtp.gmail.com
export SMTP_PORT=587
export SMTP_USER=your-email@gmail.com
export SMTP_PASSWORD=your-app-password
export SMTP_FROM=your-email@gmail.com

# Run the sender service
cd services/sender
make run
```

**Note**: For security, consider using a `.env` file (see [`.env.example`](../.env.example) in the sender directory) or a secrets manager instead of exporting credentials directly.

## Important Notes

### Gmail App Passwords

If 2-Step Verification is enabled on your Gmail account, you **must** use an App Password instead of your regular password:

1. Go to your [Google Account](https://myaccount.google.com/)
2. Navigate to **Security** → **2-Step Verification**
3. Scroll down to **App passwords**
4. Generate a new app password for "Mail"
5. Use the generated 16-character password (no spaces) as `SMTP_PASSWORD`

### Port Options

- **Port 587** (recommended): Uses STARTTLS - automatically upgrades plain connection to TLS
- **Port 465**: Uses SSL/TLS from the start - also supported

### Security

- Never commit passwords to version control
- Use environment variables or a secure secrets manager
- Consider using a dedicated Gmail account for sending alerts

## Testing

After configuration, test by:

1. Creating a rule with an email endpoint
2. Generating an alert that matches the rule
3. Checking that the email is sent successfully

## Troubleshooting

### Emails sent but not received

If logs show "Successfully sent email notification" but emails don't appear:

1. **Check Spam Folder**: Gmail may filter emails to spam, especially for new sender accounts
2. **Check Gmail Sent Folder**: Emails should appear in the sender's "Sent" folder if successfully sent
3. **FROM Address**: Gmail requires the FROM address to match the authenticated user. The service automatically uses `SMTP_USER` as the FROM address for Gmail
4. **Wait a few minutes**: Gmail may delay delivery for new accounts or unusual sending patterns
5. **Check Gmail Activity**: Go to https://myaccount.google.com/security → Recent security activity to see if Gmail blocked the login

### "Authentication failed" error

- Verify you're using an App Password (not regular password) if 2FA is enabled
- Check that `SMTP_USER` matches the email address exactly
- Ensure `SMTP_PASSWORD` has no extra spaces
- Make sure "Less secure app access" is enabled (if not using App Password)

### "Connection refused" error

- Verify Gmail SMTP is accessible from your network
- Check firewall settings
- Try port 465 instead of 587

### "TLS handshake failed"

- Ensure your system's time is correct
- Check that port 587/465 is not blocked
- Verify the SMTP_HOST is correct (`smtp.gmail.com`)

### "Failed to set sender" error

- Gmail requires the FROM address to match the authenticated user
- The service automatically uses `SMTP_USER` as FROM for Gmail
- Ensure `SMTP_FROM` matches `SMTP_USER` when using Gmail
