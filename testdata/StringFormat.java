public class StringFormat {
    public static void main(String[] args) {
        // Basic string operations without String.format
        int x = 42;
        double pi = 3.14;
        String name = "World";

        // String concatenation with various types
        System.out.println("x=" + x);           // x=42
        System.out.println("pi=" + pi);          // pi=3.14
        System.out.println("Hello " + name + "!"); // Hello World!

        // String.valueOf
        System.out.println(String.valueOf(true));  // true
        System.out.println(String.valueOf('A'));   // A
        System.out.println(String.valueOf(100));   // 100

        // Integer operations
        System.out.println(Integer.parseInt("123"));     // 123
        System.out.println(Integer.toString(456));       // 456
        System.out.println(Integer.MAX_VALUE);           // 2147483647
    }
}
