# Message Service Setup

The Message Service can send two types of messages: `EMAIL` and `CHAT`. Email is sent using AWS Simple Email Service
(SES) and chat messages are sent using Google Hangouts Chat. Both of these services must be provisioned and configured
in order to use them. Although it can be used for other things, the message service is used primarily for sending
password reset tokens (email) and runtime system notifications (chat).

To provision AWS SES, you
can [verify your domain](https://us-west-2.console.aws.amazon.com/ses/home?region=us-west-2#verified-senders-domain:).
If you're using AWS Route 53
to [manage your domain](https://console.aws.amazon.com/route53/home?region=us-west-2#hosted-zones:),
then this is a trivial exercise, and happens very quickly.

To provision a Google Chat Space, follow this procedure:

1. Log into [Google Chat](https://chat.google.com/)
1. Create a "room" (space) from the "Find people, rooms, bots" drop-down menu
1. Configure webhooks from the room title's drop-down menu
1. Add a webook, providing a title and icon. Copy the webhook URL.
1. Create an [AWS Parameter](https://us-west-2.console.aws.amazon.com/systems-manager/parameters?region=us-west-2),
   storing a portion of the URL that includes the space ID, key, and token as a Secure String. The fragment follows this
   pattern: `{spaceId}/messages?key={key}&token={token}`. Two parameter names, `/versionary/chat`
   and `/versionary/chat-test` are hard-coded in the GoogleChatClient. If you don't use those names, then of course
   you'll need to change the code.
