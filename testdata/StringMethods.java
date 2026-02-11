public class StringMethods {
    public static void main(String[] args) {
        String s = "Hello, World!";
        System.out.println(s.length());           // 13
        System.out.println(s.charAt(0));           // H
        System.out.println(s.substring(7));        // World!
        System.out.println(s.substring(0, 5));     // Hello
        System.out.println(s.indexOf("World"));    // 7
        System.out.println(s.contains("World"));   // true
        System.out.println(s.equals("Hello, World!")); // true
        System.out.println(s.equals("other"));     // false
        System.out.println(s.toUpperCase());       // HELLO, WORLD!
        System.out.println(s.toLowerCase());       // hello, world!
        System.out.println(s.trim());              // Hello, World!
        System.out.println("  hi  ".trim());       // hi
        System.out.println(s.replace(',', ';'));    // Hello; World!
        System.out.println(String.valueOf(42));     // 42
        System.out.println(s.isEmpty());            // false
        System.out.println("".isEmpty());           // true
    }
}
