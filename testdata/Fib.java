import java.util.HashMap;

class Fib {
  private HashMap<Integer, Integer> cache = new HashMap<>();

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
    new Main().fib(10);
  }
}
