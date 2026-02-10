public class ControlFlow {
    public static int abs(int x) {
        if (x < 0) {
            return -x;
        }
        return x;
    }
    public static int factorial(int n) {
        int result = 1;
        int i = 1;
        while (i <= n) {
            result = result * i;
            i = i + 1;
        }
        return result;
    }
    public static void main(String[] args) {
        System.out.println(abs(-5));      // 5
        System.out.println(abs(3));       // 3
        System.out.println(factorial(5)); // 120
    }
}
