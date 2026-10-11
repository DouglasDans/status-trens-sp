import { subscribeToTopic, unsubscribeFromTopic } from '@react-native-firebase/messaging';

import { subscribeLine, topicFor, unsubscribeLine } from './topics';

describe('topics', () => {
  it('names one topic per line', () => {
    expect(topicFor(9)).toBe('linha-9');
    expect(topicFor(17)).toBe('linha-17');
  });

  it('subscribes to the line topic', async () => {
    await subscribeLine(9);

    expect(subscribeToTopic).toHaveBeenCalledWith(expect.anything(), 'linha-9');
  });

  it('unsubscribes from the line topic', async () => {
    await unsubscribeLine(9);

    expect(unsubscribeFromTopic).toHaveBeenCalledWith(expect.anything(), 'linha-9');
  });

  it('never fails the caller when Firebase does', async () => {
    jest.mocked(subscribeToTopic).mockRejectedValueOnce(new Error('offline'));
    jest.mocked(unsubscribeFromTopic).mockRejectedValueOnce(new Error('offline'));

    await expect(subscribeLine(9)).resolves.toBeUndefined();
    await expect(unsubscribeLine(9)).resolves.toBeUndefined();
  });
});
