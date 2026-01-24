# Stripe Configuration - Next Steps

✅ **Your Stripe API key has been configured!**

Your live Stripe secret key has been added to your `.env` file:
```
STRIPE_SECRET_KEY=sk_live_YOUR_STRIPE_SECRET_KEY_HERE
```

## Required Next Steps

### 1. Create Products and Prices in Stripe Dashboard

1. Go to [Stripe Dashboard](https://dashboard.stripe.com) → **Products**
2. Create 3 products:
   - **Observer** - Basic tier
   - **Supporter** - Mid tier  
   - **Commander** - Premium tier
3. For each product, create a **Recurring** price (monthly subscription)
4. Copy the **Price ID** for each (starts with `price_`)

### 2. Update PlanPriceMap

Edit `internal/services/stripe.go` and replace the placeholder price IDs:

```go
var PlanPriceMap = map[string]string{
    "plan_observer":  "price_YOUR_OBSERVER_PRICE_ID",   // Replace with actual price ID
    "plan_supporter": "price_YOUR_SUPPORTER_PRICE_ID",  // Replace with actual price ID
    "plan_commander": "price_YOUR_COMMANDER_PRICE_ID",  // Replace with actual price ID
}
```

### 3. Set Up Webhooks

1. In Stripe Dashboard → **Developers** → **Webhooks**
2. Click **Add endpoint**
3. Enter your webhook URL: `https://yourdomain.com/api/webhooks/stripe`
   - For local testing, use a tool like [ngrok](https://ngrok.com) to expose your local server
4. Select these events:
   - ✅ `checkout.session.completed`
   - ✅ `customer.subscription.updated`
   - ✅ `customer.subscription.deleted`
   - ✅ `invoice.payment_succeeded`
   - ✅ `invoice.payment_failed`
5. Copy the **Signing secret** (starts with `whsec_`)
6. Add to `.env`:
   ```
   STRIPE_WEBHOOK_SECRET=whsec_YOUR_WEBHOOK_SECRET
   ```

### 4. Configure Redirect URLs

Add these to your `.env` file (adjust URLs for your domain):

```
STRIPE_SUCCESS_URL=https://yourdomain.com/dashboard?success=true
STRIPE_CANCEL_URL=https://yourdomain.com/pricing
STRIPE_PORTAL_RETURN_URL=https://yourdomain.com/dashboard
```

For local development:
```
STRIPE_SUCCESS_URL=http://localhost:5173/dashboard?success=true
STRIPE_CANCEL_URL=http://localhost:5173/pricing
STRIPE_PORTAL_RETURN_URL=http://localhost:5173/dashboard
```

## Testing

### Verify Configuration

The Stripe service will now:
- ✅ Use your live API key
- ✅ Return errors if key is missing (no more mock fallback)
- ✅ Process real payments
- ✅ Handle webhooks properly

### Test Flow

1. **Create a checkout session** via API:
   ```bash
   POST /api/subscriptions/checkout
   {
     "planId": "plan_observer"
   }
   ```

2. **Complete payment** in Stripe test mode first (use test keys for testing)

3. **Verify webhook** is received and processed

4. **Check database** for subscription record

## Important Notes

⚠️ **Security:**
- Your `.env` file should be in `.gitignore` (it is)
- Never commit API keys to version control
- Use test keys (`sk_test_...`) for development
- Rotate keys if exposed

⚠️ **Production:**
- Use live keys (`sk_live_...`) only in production
- Set up proper webhook endpoint with HTTPS
- Monitor webhook delivery in Stripe Dashboard
- Set up error alerts for failed payments

## Support

- See `Documentation/Stripe_Setup_Guide.md` for detailed instructions
- [Stripe Documentation](https://stripe.com/docs)
- [Stripe API Reference](https://stripe.com/docs/api)
