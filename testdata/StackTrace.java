public class StackTrace {
    static int factorial(int n) {
        if (n < 0) throw new IllegalArgumentException("negative: " + n);
        if (n <= 1) return 1;
        return n * factorial(n - 1);
    }

    public static void main(String[] args) {
        System.out.println(factorial(5));  // 120
        try {
            factorial(-1);
        } catch (IllegalArgumentException e) {
            System.out.println(e.getMessage()); // negative: -1
        }
    }
}
