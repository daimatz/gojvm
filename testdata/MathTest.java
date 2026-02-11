public class MathTest {
    public static void main(String[] args) {
        // Math.abs
        System.out.println(Math.abs(-42));   // 42
        System.out.println(Math.abs(10));    // 10

        // Math.max, Math.min
        System.out.println(Math.max(3, 7));  // 7
        System.out.println(Math.min(3, 7));  // 3

        // Math.pow (int cast)
        System.out.println((int) Math.pow(2, 10)); // 1024

        // Pythagorean theorem: 3-4-5 triangle
        double a = 3.0;
        double b = 4.0;
        double c = Math.sqrt(a * a + b * b);
        System.out.println((int) c);         // 5

        // Math.floor, Math.ceil
        System.out.println((int) Math.floor(3.7));  // 3
        System.out.println((int) Math.ceil(3.2));    // 4
    }
}
