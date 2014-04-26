package ch.andrewreus.kvo;

public class KvoObject {
	protected abstract class KvoProperty<T> {
		T value_;
		boolean isUpdating_;
		
		public KvoProperty(T initialValue) {
			value_ = initialValue;
		}
		
		public T get() {
			return value_;
		}
		
		void update(T value) {
			if (isUpdating_) {
				throw new IllegalStateException("Alreadying updating the given value!");
			}
			isUpdating_ = true;
			T old = value;
			value_ = value;
			notifySubscribers(old);
			isUpdating_ = false;
		}
		
		protected abstract void notifySubscribers(T old);
	}
}
