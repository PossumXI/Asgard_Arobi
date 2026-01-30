# Stripe Integration Setup Guide

This guide walks you through setting up Stripe payments for ASGARD.

## Quick Start

### 1. Set Your Stripe API Key

**Option A: Using PowerShell Script (Recommended)**
```powershell
cd scripts
.\setup_stripe.ps1 -StripeSecretKey "sk_live_YOUR_STRIPE_SECRET_KEY_HERE"
```

**Option B: Manual Environment Variable**
```powershell
# Windows PowerShell
$env:STRIPE_SECRET_KEY="sk_live_YOUR_STRIPE_SECRET_KEY_HERE"

# Or add to .env file
STRIPE_SECRET_KEY=sk_live_YOUR_STRIPE_SECRET_KEY_HERE
```

### 2. Configure Stripe Price IDs

1. Log into [Stripe Dashboard](https://dashboard.stripe.com)
2. Go to **Products** → Create products for each tier:
   - **Observer** tier
   - **Supporter** tier  
   - **Commander** tier
3. Create prices for each product (monthly subscription)
4. Copy the Price IDs (start with `price_`)
5. Update `internal/services/stripe.go`:

```go
var PlanPriceMap = map[string]string{
    "plan_observer":  "price_YOUR_OBSERVER_PRICE_ID",
    "plan_supporter": "price_YOUR_SUPPORTER_PRICE_ID",
    "plan_commander": "price_YOUR_COMMANDER_PRICE_ID",
}
```

### 3. Set Up Webhooks

1. In Stripe Dashboard → **Developers** → **Webhooks**
2. Click **Add endpoint**
3. Enter endpoint URL: `https://aura-genesis.org/api/webhooks/stripe`
4. Select these events:
   - `checkout.session.completed`
   - `customer.subscription.updated`
   - `customer.subscription.deleted`
   - `invoice.payment_succeeded`
   - `invoice.payment_failed`
5. Copy the **Signing secret** (starts with `whsec_`)
6. Set it as environment variable:

```powershell
$env:STRIPE_WEBHOOK_SECRET="whsec_YOUR_WEBHOOK_SECRET"
```

### 4. Configure Redirect URLs

Set these environment variables for checkout redirects:

```powershell
$env:STRIPE_SUCCESS_URL="https://aura-genesis.org/dashboard?success=true"
$env:STRIPE_CANCEL_URL="https://aura-genesis.org/pricing"
$env:STRIPE_PORTAL_RETURN_URL="https://aura-genesis.org/dashboard"
```

## Verification

### Test Stripe Connection

The service will validate that Stripe is configured:

- ✅ If `STRIPE_SECRET_KEY` is set: Operations proceed normally
- ❌ If `STRIPE_SECRET_KEY` is missing: Returns error "stripe is not configured"

### Check Logs

Look for these messages:
- `"stripe is not configured"` - API key missing
- `"Failed to create checkout session"` - Check API key validity
- `"Webhook signature verification failed"` - Check webhook secret

## Testing

### Test Mode

For testing, use test keys:
- Test secret key: `sk_test_...`
- Test webhook secret: `whsec_test_...`

### Test Checkout Flow

1. Create a checkout session via API
2. Complete payment in Stripe test mode
3. Verify webhook is received
4. Check database for subscription record

## Production Checklist

- [ ] Use live API keys (`sk_live_...`)
- [ ] Configure webhook endpoint with HTTPS
- [ ] Set up webhook signing secret
- [ ] Configure all redirect URLs
- [ ] Set up Price IDs for all tiers
- [ ] Test checkout flow end-to-end
- [ ] Monitor webhook delivery in Stripe Dashboard
- [ ] Set up error alerts for failed payments

## Troubleshooting

### "stripe is not configured" Error

**Cause:** `STRIPE_SECRET_KEY` environment variable not set

**Solution:**
```powershell
$env:STRIPE_SECRET_KEY="sk_live_YOUR_KEY"
# Restart application
```

### Webhook Not Received

**Check:**
1. Webhook endpoint is publicly accessible (HTTPS)
2. Webhook secret matches Stripe Dashboard
3. Events are selected in Stripe Dashboard
4. Check Stripe Dashboard → Webhooks → Recent events

### Payment Succeeded But Subscription Not Created

**Check:**
1. Webhook is being received (Stripe Dashboard)
2. Webhook handler logs for errors
3. Database connection is working
4. Repository methods are working correctly

## Security Notes

⚠️ **IMPORTANT:**
- Never commit API keys to version control
- Use environment variables or secure secret management
- Rotate keys if exposed
- Use test keys for development
- Monitor Stripe Dashboard for suspicious activity

## Support

- [Stripe Documentation](https://stripe.com/docs)
- [Stripe API Reference](https://stripe.com/docs/api)
- [Stripe Webhooks Guide](https://stripe.com/docs/webhooks)
