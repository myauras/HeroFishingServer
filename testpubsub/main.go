const { PubSub } = require('@google-cloud/pubsub');

const projectId = 'aurafortest';
const topicName = 'herofishing-json-topic';
const subscriptionName = 'herofishing-subscription';

const pubsub = new PubSub({ projectId });
const topic = pubsub.topic(topicName);
const subscription = topic.subscription(subscriptionName);

const fetchJsonFromSubscription = async () => {
    return new Promise(async (resolve, reject) => {
        // Check if the subscription already exists
        const [subscriptions] = await pubsub.getSubscriptions();
        const exists = subscriptions.some(sub => sub.name.endsWith(subscriptionName));

        if (!exists) {
            try {
                await topic.createSubscription(subscriptionName);
            } catch (error) {
                if (error.code !== 6) {  // ALREADY_EXISTS
                    reject(`Failed to create subscription: ${error}`);
                    return;
                }
            }
        }

        // Receive a single message from the subscription
        const messageHandler = (message) => {
            resolve(JSON.parse(message.data.toString()));
            message.ack();
            subscription.removeListener('message', messageHandler);
        };

        subscription.on('message', messageHandler);

        // Handle any errors that occur while receiving messages
        subscription.on('error', error => {
            reject(`Failed to receive messages: ${error}`);
        });
    });
};

module.exports = fetchJsonFromSubscription;
