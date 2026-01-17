# Email Sending Troubleshooting Guide

## Problem: Logs show "Successfully sent" but emails don't appear

If your logs show `Successfully sent email notification` but emails don't appear in inbox or sent folder, follow these steps:

### Step 1: Enable Debug Logging

Run the sender with debug logging to see detailed SMTP conversation:

```bash
export LOG_LEVEL=DEBUG
cd services/sender
make run
```

This will show:
- Authentication steps
- Each SMTP command (MAIL, RCPT, DATA)
- Any errors during the SMTP transaction

### Step 2: Check Gmail Security Settings

Gmail may be blocking the emails. Check:

1. **Gmail Security Activity**:
   - Go to https://myaccount.google.com/security
   - Click "Recent security activity"
   - Look for blocked login attempts or suspicious activity

2. **App Password**:
   - Ensure you're using an App Password (not regular password)
   - If 2FA is enabled, you MUST use App Password
   - Generate new App Password: Security → 2-Step Verification → App passwords

3. **"Less secure app access"** (if not using App Password):
   - This is deprecated but some accounts may still need it
   - Go to: https://myaccount.google.com/lesssecureapps

### Step 3: Test SMTP Connection Manually

Test if SMTP connection works using `telnet` or `openssl`:

```bash
# Test STARTTLS on port 587
openssl s_client -connect smtp.gmail.com:587 -starttls smtp

# Or test SSL on port 465
openssl s_client -connect smtp.gmail.com:465
```

You should see Gmail's SMTP greeting. Type `QUIT` to exit.

### Step 4: Verify Email Headers

The service now includes proper email headers:
- `Date` header
- `MIME-Version`
- `Content-Type`
- `Content-Transfer-Encoding`

Check the debug logs to see the full email message being sent.

### Step 5: Check Gmail Filters

1. Check **Spam folder** in recipient's Gmail
2. Check **All Mail** folder (not just Inbox)
3. Check if there are any **filters** that might be moving emails
4. Check **Gmail Sent folder** of the sender account

### Step 6: Test with a Simple Email

Try sending a test email to yourself first:

```bash
# Set environment variables
export SMTP_HOST=smtp.gmail.com
export SMTP_PORT=587
export SMTP_USER=alert.system.notify.email@gmail.com
export SMTP_PASSWORD=AlertsystemnotifyemailPassword123
export SMTP_FROM=alert.system.notify.email@gmail.com
export LOG_LEVEL=DEBUG

# Run sender and trigger a notification
cd services/sender
make run
```

### Step 7: Check Gmail Account Status

1. **Account Status**: Ensure the Gmail account is active and not suspended
2. **Sending Limits**: Gmail has daily sending limits (500 emails/day for free accounts)
3. **Account Age**: Very new accounts may have stricter limits

### Step 8: Alternative - Use MailHog for Testing

If Gmail continues to have issues, test with MailHog first to verify the code works:

```bash
# Start MailHog (if not already running)
docker compose up -d mailhog

# Configure sender to use MailHog
export SMTP_HOST=localhost
export SMTP_PORT=1025
export SMTP_FROM=test@example.com
# No SMTP_USER/SMTP_PASSWORD needed

# View emails at http://localhost:8025
```

If emails appear in MailHog but not Gmail, the issue is with Gmail configuration, not the code.

## Common Issues

### "SMTP authentication failed"
- Wrong password (use App Password if 2FA enabled)
- Username doesn't match email address
- Account security settings blocking access

### "Failed to close data writer"
- Gmail rejected the email content
- Check email headers and format
- May indicate spam filtering

### Emails in Spam
- New sender account
- Missing or incorrect email headers
- Content triggers spam filters
- Solution: Mark as "Not Spam" and Gmail will learn

### No emails in Sent folder
- Gmail rejected after accepting DATA command
- Check Gmail security activity
- May need to wait a few minutes

## Still Not Working?

1. **Check Gmail Activity**: https://myaccount.google.com/security → Recent security activity
2. **Try Different Port**: Switch from 587 to 465 (or vice versa)
3. **Verify Credentials**: Double-check SMTP_USER and SMTP_PASSWORD
4. **Test with MailHog**: Verify the code works with local SMTP
5. **Check Firewall**: Ensure ports 587/465 are not blocked

## Getting Help

If issues persist:
1. Enable `LOG_LEVEL=DEBUG` and capture full logs
2. Check Gmail security activity page
3. Verify App Password is correct
4. Test with MailHog to isolate the issue
