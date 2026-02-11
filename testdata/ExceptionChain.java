public class ExceptionChain {
    // Test multiple exception types and exception hierarchy
    static String test(int n) {
        try {
            if (n == 0) {
                throw new IllegalArgumentException("zero");
            }
            if (n < 0) {
                throw new RuntimeException("negative");
            }
            return "ok:" + (100 / n);
        } catch (IllegalArgumentException e) {
            return "illegal:" + e.getMessage();
        } catch (RuntimeException e) {
            return "runtime:" + e.getMessage();
        }
    }

    public static void main(String[] args) {
        System.out.println(test(5));   // ok:20
        System.out.println(test(0));   // illegal:zero
        System.out.println(test(-1));  // runtime:negative

        // Nested try-catch with finally
        try {
            try {
                throw new RuntimeException("inner");
            } finally {
                System.out.println("finally1"); // finally1
            }
        } catch (RuntimeException e) {
            System.out.println(e.getMessage()); // inner
        }

        // ArithmeticException caught
        try {
            int x = 10 / 0;
            System.out.println("unreachable");
        } catch (ArithmeticException e) {
            System.out.println("caught"); // caught
        }
    }
}
