import java.util.HashMap;

class Fib {
  private HashMap<Integer, Integer> cache;

  public Fib() {
    cache = new HashMap<>();
    cache.put(0, 1);
    cache.put(1, 1);
  }
  public int fib(int n) {
    Integer got = cache.get(n);
    if (got != null) {
      return got;
    }
    Integer value = fib(n-1) + fib(n-2);
    cache.put(n, value);
    return value;
  }
  public static void main(String[] args) {
    System.out.println(new Fib().fib(10));
  }
}
