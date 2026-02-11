public class StringBuilderTest {
    public static void main(String[] args) {
        // Basic StringBuilder usage
        StringBuilder sb = new StringBuilder();
        sb.append("Hello");
        sb.append(" ");
        sb.append("World");
        System.out.println(sb.toString()); // Hello World

        // Append different types
        StringBuilder sb2 = new StringBuilder();
        sb2.append("n=");
        sb2.append(42);
        sb2.append(",pi=");
        sb2.append(3.14);
        System.out.println(sb2.toString()); // n=42,pi=3.14

        // Length and charAt
        String s = sb.toString();
        System.out.println(s.length()); // 11
        System.out.println(s.charAt(0)); // H

        // Chained append
        String result = new StringBuilder()
            .append("a")
            .append("b")
            .append("c")
            .toString();
        System.out.println(result); // abc
    }
}
