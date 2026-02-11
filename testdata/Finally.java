public class Finally {
    public static void main(String[] args) {
        // Test 1: finally runs after try
        try {
            System.out.println(1);
        } finally {
            System.out.println(2);
        }
        // Test 2: finally runs after catch
        try {
            int x = 1 / 0;
        } catch (ArithmeticException e) {
            System.out.println(3);
        } finally {
            System.out.println(4);
        }
        // Test 3: finally with normal return
        System.out.println(safeValue());
    }

    static int safeValue() {
        try {
            return 10;
        } finally {
            System.out.println(5);
        }
    }
}
